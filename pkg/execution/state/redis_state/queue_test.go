package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const testPriority = PriorityDefault

func TestQueueEnqueueItem(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.ULID{})

		// Ensure that our data is set up correctly.
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure the partition is inserted.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        start.Unix(),
		}, qp)
	})

	t.Run("It sets the right item score", func(t *testing.T) {
		start := time.Now()
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		requireItemScoreEquals(t, r, item, start)
	})

	t.Run("It enqueues an item in the future", func(t *testing.T) {
		at := time.Now().Add(time.Hour).Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is still
		// the start time.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        start.Unix(),
		}, qp)

		// Ensure that the zscore did not change.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(defaultQueueKey.PartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, start.Unix(), score)
	})

	t.Run("Updates partition vesting time to earlier times", func(t *testing.T) {
		at := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{}, at)
		require.NoError(t, err)

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        at.Unix(),
		}, qp)

		// Assert that the zscore was changed to this earliest timestamp.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))
		score, err := r.ZScore(defaultQueueKey.PartitionIndex(), keys[0])
		require.NoError(t, err)
		require.EqualValues(t, at.Unix(), score)
	})

	t.Run("Adding another workflow ID increases partition set", func(t *testing.T) {
		at := time.Now().Truncate(time.Second)
		item, err := q.EnqueueItem(ctx, QueueItem{
			WorkflowID: uuid.New(),
		}, at)
		require.NoError(t, err)

		// Assert that we have two zscores in partition:sorted.
		keys, err := r.ZMembers(defaultQueueKey.PartitionIndex())
		require.NoError(t, err)
		require.Equal(t, 2, len(keys))

		// Ensure the partition is inserted, and the earliest time is updated
		// inside the partition item.
		qp := getPartition(t, r, item.WorkflowID)
		require.Equal(t, QueuePartition{
			WorkflowID: item.WorkflowID,
			Priority:   testPriority,
			AtS:        at.Unix(),
		}, qp)
	})
}

func TestQueueEnqueueItemIdempotency(t *testing.T) {
	dur := 2 * time.Second

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	// Set idempotency to a second
	q := NewQueue(rc, WithIdempotencyTTL(dur))
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)

	t.Run("It enqueues an item only once", func(t *testing.T) {
		i := QueueItem{ID: "once"}

		item, err := q.EnqueueItem(ctx, i, start)
		require.NoError(t, err)
		require.Equal(t, hashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found := getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)

		// Ensure we can't enqueue again.
		_, err = q.EnqueueItem(ctx, i, start)
		require.Equal(t, ErrQueueItemExists, err)

		// Dequeue
		err = q.Dequeue(ctx, item)
		require.NoError(t, err)

		// Ensure we can't enqueue even after dequeue.
		_, err = q.EnqueueItem(ctx, i, start)
		require.Equal(t, ErrQueueItemExists, err)

		// Wait for the idempotency TTL to expire
		r.FastForward(dur)

		item, err = q.EnqueueItem(ctx, i, start)
		require.NoError(t, err)
		require.Equal(t, hashID(ctx, "once"), item.ID)
		require.NotEqual(t, i.ID, item.ID)
		found = getQueueItem(t, r, item.ID)
		require.Equal(t, item, found)
	})
}

func TestQueuePeek(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	t.Run("It returns none with no items enqueued", func(t *testing.T) {
		items, err := q.Peek(ctx, uuid.UUID{}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 0, len(items))
	})

	t.Run("It returns an ordered list of items", func(t *testing.T) {
		a := time.Now().Truncate(time.Second)
		b := a.Add(2 * time.Second)
		c := b.Add(2 * time.Second)
		d := c.Add(2 * time.Second)

		ia, err := q.EnqueueItem(ctx, QueueItem{}, a)
		require.NoError(t, err)
		ib, err := q.EnqueueItem(ctx, QueueItem{}, b)
		require.NoError(t, err)
		ic, err := q.EnqueueItem(ctx, QueueItem{}, c)
		require.NoError(t, err)

		items, err := q.Peek(ctx, uuid.UUID{}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 3, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		require.NotEqualValues(t, []*QueueItem{&ib, &ia, &ic}, items)

		id, err := q.EnqueueItem(ctx, QueueItem{}, d)
		require.NoError(t, err)

		items, err = q.Peek(ctx, uuid.UUID{}, time.Now().Add(time.Hour), 10)
		require.NoError(t, err)
		require.EqualValues(t, 4, len(items))
		require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)

		t.Run("It should limit the list", func(t *testing.T) {
			items, err = q.Peek(ctx, uuid.UUID{}, time.Now().Add(time.Hour), 2)
			require.NoError(t, err)
			require.EqualValues(t, 2, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib}, items)
		})

		t.Run("It should apply a peek offset", func(t *testing.T) {
			items, err = q.Peek(ctx, uuid.UUID{}, time.Now().Add(-1*time.Hour), QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))

			items, err = q.Peek(ctx, uuid.UUID{}, c, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic}, items)
		})

		t.Run("It should remove any leased items from the list", func(t *testing.T) {
			// Lease step B, and it should be removed.
			leaseID, err := q.Lease(ctx, ia.WorkflowID, ia.ID, 50*time.Millisecond)
			require.NoError(t, err)

			items, err = q.Peek(ctx, uuid.UUID{}, d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 3, len(items))
			require.EqualValues(t, []*QueueItem{&ib, &ic, &id}, items)

			// When the lease expires it should re-appear
			<-time.After(52 * time.Millisecond)

			items, err = q.Peek(ctx, uuid.UUID{}, d, QueuePeekMax)
			require.NoError(t, err)
			require.EqualValues(t, 4, len(items))
			ia.LeaseID = leaseID
			// NOTE: item A should have an expired lease ID.
			require.EqualValues(t, []*QueueItem{&ia, &ib, &ic, &id}, items)
		})
	})

}

func TestQueueLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		now := time.Now()
		id, err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It should increase the in-progress count", func(t *testing.T) {
			val := r.HGet(defaultQueueKey.PartitionMeta(item.WorkflowID.String()), "n")
			require.NotEmpty(t, val)
			require.Equal(t, "1", val)
		})

		t.Run("Leasing again should fail", func(t *testing.T) {
			for i := 0; i < 50; i++ {
				id, err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
				require.Equal(t, ErrQueueItemAlreadyLeased, err)
				require.Nil(t, id)
				<-time.After(5 * time.Millisecond)
			}
		})

		t.Run("Leasing an expired lease should succeed", func(t *testing.T) {
			<-time.After(1005 * time.Millisecond)
			now := time.Now()
			id, err := q.Lease(ctx, item.WorkflowID, item.ID, 5*time.Second)
			require.NoError(t, err)
			require.NoError(t, err)

			item = getQueueItem(t, r, item.ID)
			require.NotNil(t, item.LeaseID)
			require.EqualValues(t, id, item.LeaseID)
			require.WithinDuration(t, now.Add(5*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

			t.Run("Expired does not increase partition in-progress count", func(t *testing.T) {
				val := r.HGet(defaultQueueKey.PartitionMeta(item.WorkflowID.String()), "n")
				require.NotEmpty(t, val)
				require.Equal(t, "1", val)
			})
		})

		t.Run("It should increase the score of the item by the lease duration", func(t *testing.T) {
			start := time.Now()
			item, err := q.EnqueueItem(ctx, QueueItem{}, start)
			require.NoError(t, err)
			require.Nil(t, item.LeaseID)

			requireItemScoreEquals(t, r, item, start)

			_, err = q.Lease(ctx, item.WorkflowID, item.ID, time.Minute)
			require.NoError(t, err)

			requireItemScoreEquals(t, r, item, start.Add(time.Minute))
		})
	})
}

func TestQueueExtendLease(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	start := time.Now().Truncate(time.Second)
	t.Run("It leases an item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		now := time.Now()
		id, err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, id, item.LeaseID)
		require.WithinDuration(t, now.Add(time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		now = time.Now()
		nextID, err := q.ExtendLease(ctx, item, *id, 10*time.Second)
		require.NoError(t, err)

		// Ensure the leased item has the next ID.
		item = getQueueItem(t, r, item.ID)
		require.NotNil(t, item.LeaseID)
		require.EqualValues(t, nextID, item.LeaseID)
		require.WithinDuration(t, now.Add(10*time.Second), ulid.Time(item.LeaseID.Time()), 20*time.Millisecond)

		t.Run("It fails with an invalid lease ID", func(t *testing.T) {
			invalid := ulid.MustNew(ulid.Now(), rnd)
			nextID, err := q.ExtendLease(ctx, item, invalid, 10*time.Second)
			require.EqualValues(t, ErrQueueItemLeaseMismatch, err)
			require.Nil(t, nextID)
		})
	})

	t.Run("It does not extend an unleased item", func(t *testing.T) {
		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)

		nextID, err := q.ExtendLease(ctx, item, ulid.ULID{}, 10*time.Second)
		require.EqualValues(t, ErrQueueItemNotLeased, err)
		require.Nil(t, nextID)

		item = getQueueItem(t, r, item.ID)
		require.Nil(t, item.LeaseID)
	})
}

func TestQueueDequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	t.Run("It should remove a queue item", func(t *testing.T) {
		start := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, start)
		require.NoError(t, err)

		id, err := q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
		require.NoError(t, err)

		err = q.Dequeue(ctx, item)
		require.NoError(t, err)

		t.Run("It should remove the item from the queue map", func(t *testing.T) {
			val := r.HGet(defaultQueueKey.QueueItem(), id.String())
			require.Empty(t, val)
		})

		t.Run("Extending a lease should fail after dequeue", func(t *testing.T) {
			id, err := q.ExtendLease(ctx, item, *id, time.Minute)
			require.Equal(t, ErrQueueItemNotFound, err)
			require.Nil(t, id)
		})

		t.Run("It should remove the item from the queue index", func(t *testing.T) {
			items, err := q.Peek(ctx, item.WorkflowID, time.Now().Add(time.Hour), 10)
			require.NoError(t, err)
			require.EqualValues(t, 0, len(items))
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			val := r.HGet(defaultQueueKey.PartitionMeta(item.WorkflowID.String()), "n")
			require.NotEmpty(t, val)
			require.Equal(t, "0", val)
		})

		t.Run("It should work if the item is not leased (eg. deletions)", func(t *testing.T) {
			item, err := q.EnqueueItem(ctx, QueueItem{}, start)
			require.NoError(t, err)

			err = q.Dequeue(ctx, item)
			require.NoError(t, err)

			val := r.HGet(defaultQueueKey.QueueItem(), id.String())
			require.Empty(t, val)
		})
	})
}

func TestQueueRequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	t.Run("Re-enqueuing a leased item should succeed", func(t *testing.T) {
		now := time.Now()

		item, err := q.EnqueueItem(ctx, QueueItem{}, now)
		require.NoError(t, err)
		_, err = q.Lease(ctx, item.WorkflowID, item.ID, time.Second)
		require.NoError(t, err)

		// Assert partition index is original
		pi := QueuePartition{WorkflowID: item.WorkflowID, Priority: testPriority}
		requirePartitionScoreEquals(t, r, pi.WorkflowID, now.Truncate(time.Second))

		requireInProgress(t, r, item.WorkflowID, 1)

		next := now.Add(time.Hour)
		err = q.Requeue(ctx, item, next)
		require.NoError(t, err)

		t.Run("It should re-enqueue the item with the future time", func(t *testing.T) {
			requireItemScoreEquals(t, r, item, next)
		})

		t.Run("It should always remove the lease from the re-enqueued item", func(t *testing.T) {
			fetched := getQueueItem(t, r, item.ID)
			require.Nil(t, fetched.LeaseID)
		})

		t.Run("It should decrease the in-progress count", func(t *testing.T) {
			requireInProgress(t, r, item.WorkflowID, 0)
		})

		t.Run("It should update the partition's earliest time, if earliest", func(t *testing.T) {
			// Assert partition index is updated, as there's only one item here.
			requirePartitionScoreEquals(t, r, pi.WorkflowID, next)
		})

		t.Run("It should not update the partition's earliest time, if later", func(t *testing.T) {
			_, err := q.EnqueueItem(ctx, QueueItem{}, now)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.WorkflowID, now)

			next := now.Add(2 * time.Hour)
			err = q.Requeue(ctx, item, next)
			require.NoError(t, err)

			requirePartitionScoreEquals(t, r, pi.WorkflowID, now)
		})
	})
}

func TestQueuePartitionLease(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idA, idB, idC := uuid.New(), uuid.New(), uuid.New()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()

	_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, atA)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idB}, atB)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, QueueItem{WorkflowID: idC}, atC)
	require.NoError(t, err)

	t.Run("Partitions are in order after enqueueing", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idA, Priority: testPriority, AtS: atA.Unix()},
			{WorkflowID: idB, Priority: testPriority, AtS: atB.Unix()},
			{WorkflowID: idC, Priority: testPriority, AtS: atC.Unix()},
		}, items)
	})

	leaseUntil := now.Add(3 * time.Second)

	t.Run("It leases a partition", func(t *testing.T) {
		// Lease the first item
		leaseID, err := q.PartitionLease(ctx, idA, time.Until(leaseUntil))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		t.Run("It updates the partition score", func(t *testing.T) {
			items, err := q.PartitionPeek(ctx, true, now.Add(time.Hour), PartitionPeekMax)

			// Require the lease ID is within 25 MS of the expected value.
			require.WithinDuration(t, leaseUntil, ulid.Time(leaseID.Time()), 25*time.Millisecond)

			require.NoError(t, err)
			require.Len(t, items, 3)
			require.EqualValues(t, []*QueuePartition{
				{WorkflowID: idB, Priority: testPriority, AtS: atB.Unix()},
				{WorkflowID: idC, Priority: testPriority, AtS: atC.Unix()},
				{
					WorkflowID: idA,
					Priority:   testPriority,
					AtS:        ulid.Time(leaseID.Time()).Unix(),
					Last:       time.Now().Unix(),
					LeaseID:    leaseID,
				}, // idA is now last.
			}, items)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
		})

		t.Run("It can't lease an existing partition lease", func(t *testing.T) {
			id, err := q.PartitionLease(ctx, idA, time.Second*29)
			require.Equal(t, ErrPartitionAlreadyLeased, err)
			require.Nil(t, id)

			// Assert that score didn't change (we added 1 second in the previous test)
			requirePartitionScoreEquals(t, r, idA, leaseUntil)
		})

	})

	t.Run("It allows leasing an expired partition lease", func(t *testing.T) {
		<-time.After(time.Until(leaseUntil))

		requirePartitionScoreEquals(t, r, idA, leaseUntil)

		id, err := q.PartitionLease(ctx, idA, time.Second*5)
		require.Nil(t, err)
		require.NotNil(t, id)

		requirePartitionScoreEquals(t, r, idA, time.Now().Add(time.Second*5))
	})
}

func TestQueuePartitionPeek(t *testing.T) {
	idA := uuid.New() // low pri
	idB := uuid.New()
	idC := uuid.New()

	newQueueItem := func(id uuid.UUID) QueueItem {
		return QueueItem{
			WorkflowID: id,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					WorkflowID: id,
				},
			},
		}
	}

	now := time.Now().Truncate(time.Second).UTC()
	atA, atB, atC := now, now.Add(time.Second), now.Add(2*time.Second)

	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(
		rc,
		WithPriorityFinder(func(ctx context.Context, qi *osqueue.Item) uint {
			switch qi.Identifier.WorkflowID {
			case idB, idC:
				return PriorityMax
			default:
				return PriorityMin // Sorry A
			}
		}),
	)
	ctx := context.Background()

	_, err := q.EnqueueItem(ctx, newQueueItem(idA), atA)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, newQueueItem(idB), atB)
	require.NoError(t, err)
	_, err = q.EnqueueItem(ctx, newQueueItem(idC), atC)
	require.NoError(t, err)

	t.Run("Sequentially returns indexes in order", func(t *testing.T) {
		items, err := q.PartitionPeek(ctx, true, time.Now().Add(time.Hour), PartitionPeekMax)
		require.NoError(t, err)
		require.Len(t, items, 3)
		require.EqualValues(t, []*QueuePartition{
			{WorkflowID: idA, Priority: PriorityMin, AtS: atA.Unix()},
			{WorkflowID: idB, Priority: PriorityMax, AtS: atB.Unix()},
			{WorkflowID: idC, Priority: PriorityMax, AtS: atC.Unix()},
		}, items)
	})

	t.Run("Random returns items randomly using weighted sample", func(t *testing.T) {
		a, b, c := 0, 0, 0
		for i := 0; i <= 1000; i++ {
			items, err := q.PartitionPeek(ctx, false, time.Now().Add(time.Hour), PartitionPeekMax)
			require.NoError(t, err)
			require.Len(t, items, 3)
			switch items[0].WorkflowID {
			case idA:
				a++
			case idB:
				b++
			case idC:
				c++
			default:
				t.Fatal()
			}
		}
		// Statistically this is going to fail at some point, but we want to ensure randomness
		// will return low priority items less.
		require.GreaterOrEqual(t, a, 1) // A may be called low-digit times.
		require.Less(t, a, 250)         // But less than 1/4 (it's 1 in 10, statistically)
		require.Greater(t, c, 300)
		require.Greater(t, b, 300)
	})
}

func TestQueuePartitionRequeue(t *testing.T) {
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(rc)
	ctx := context.Background()
	idA := uuid.New()
	now := time.Now()

	qi, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	t.Run("Doesn't requeue the partition if there's an unleased job", func(t *testing.T) {
		requirePartitionScoreEquals(t, r, idA, now)
		next := now.Add(time.Hour)
		err := q.PartitionRequeue(ctx, idA, next)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, now)
	})

	t.Run("Requeus the partition with a leased job", func(t *testing.T) {
		_, err := q.Lease(ctx, idA, qi.ID, 10*time.Second)
		require.NoError(t, err)

		requirePartitionScoreEquals(t, r, idA, now)
		next := now.Add(time.Hour)
		err = q.PartitionRequeue(ctx, idA, next)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, next)
	})

	t.Run("It removes any lease when requeueing", func(t *testing.T) {
		next := now.Add(5 * time.Second)

		_, err := q.PartitionLease(ctx, idA, time.Minute)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, idA, next)
		require.NoError(t, err)
		requirePartitionScoreEquals(t, r, idA, next)

		loaded := getPartition(t, r, idA)
		require.Nil(t, loaded.LeaseID)
	})

	t.Run("It removes the partition if there are no jobs available", func(t *testing.T) {
		err := q.Dequeue(ctx, qi)
		require.NoError(t, err)

		err = q.PartitionRequeue(ctx, idA, time.Now().Add(time.Minute))
		require.Equal(t, ErrPartitionGarbageCollected, err)
	})
}

func TestQueuePartitionReprioritize(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	idA := uuid.New()

	priority := PriorityMin
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	defer rc.Close()
	q := NewQueue(
		rc,
		WithPriorityFinder(func(ctx context.Context, item *osqueue.Item) uint {
			return priority
		}),
	)
	ctx := context.Background()

	_, err := q.EnqueueItem(ctx, QueueItem{WorkflowID: idA}, now)
	require.NoError(t, err)

	first := getPartition(t, r, idA)
	require.Equal(t, first.Priority, PriorityMin)

	t.Run("It updates priority", func(t *testing.T) {
		priority = PriorityMax
		err = q.PartitionReprioritize(ctx, idA, PriorityMax)
		require.NoError(t, err)
		second := getPartition(t, r, idA)
		require.Equal(t, second.Priority, PriorityMax)
	})

	t.Run("It doesn't accept min priorities", func(t *testing.T) {
		err = q.PartitionReprioritize(ctx, idA, PriorityMin+1)
		require.Equal(t, ErrPriorityTooLow, err)
	})
}

func TestQueueLeaseSequential(t *testing.T) {
	ctx := context.Background()
	r := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: r.Addr(), PoolSize: 100})
	q := queue{
		kg: defaultQueueKey,
		r:  rc,
		pf: func(ctx context.Context, item *osqueue.Item) uint {
			return PriorityMin
		},
	}

	var (
		leaseID *ulid.ULID
		err     error
	)

	t.Run("It claims sequential leases", func(t *testing.T) {
		now := time.Now()
		dur := 500 * time.Millisecond
		leaseID, err = q.LeaseSequential(ctx, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It doesn't allow leasing without an existing lease ID", func(t *testing.T) {
		id, err := q.LeaseSequential(ctx, time.Second)
		require.Equal(t, ErrSequentialAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It doesn't allow leasing with an invalid lease ID", func(t *testing.T) {
		newULID := ulid.MustNew(ulid.Now(), rnd)
		id, err := q.LeaseSequential(ctx, time.Second, &newULID)
		require.Equal(t, ErrSequentialAlreadyLeased, err)
		require.Nil(t, id)
	})

	t.Run("It extends the lease with a valid lease ID", func(t *testing.T) {
		require.NotNil(t, leaseID)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.LeaseSequential(ctx, dur, leaseID)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})

	t.Run("It allows leasing when the current lease is expired", func(t *testing.T) {
		<-time.After(100 * time.Millisecond)

		now := time.Now()
		dur := 50 * time.Millisecond
		leaseID, err = q.LeaseSequential(ctx, dur)
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.WithinDuration(t, now.Add(dur), ulid.Time(leaseID.Time()), 5*time.Millisecond)
	})
}

func getQueueItem(t *testing.T, r *miniredis.Miniredis, id string) QueueItem {
	t.Helper()
	// Ensure that our data is set up correctly.
	val := r.HGet(defaultQueueKey.QueueItem(), id)
	require.NotEmpty(t, val)
	i := QueueItem{}
	err := json.Unmarshal([]byte(val), &i)
	require.NoError(t, err)
	return i
}

func requireInProgress(t *testing.T, r *miniredis.Miniredis, workflowID uuid.UUID, count int) {
	t.Helper()
	val := r.HGet(defaultQueueKey.PartitionMeta(workflowID.String()), "n")
	require.NotEmpty(t, val)
	require.Equal(t, fmt.Sprintf("%d", count), val)
}

func getPartition(t *testing.T, r *miniredis.Miniredis, id uuid.UUID) QueuePartition {
	t.Helper()
	val := r.HGet(defaultQueueKey.PartitionItem(), id.String())
	qp := QueuePartition{}
	err := json.Unmarshal([]byte(val), &qp)
	require.NoError(t, err)
	return qp
}

func requireItemScoreEquals(t *testing.T, r *miniredis.Miniredis, item QueueItem, expected time.Time) {
	t.Helper()
	score, err := r.ZScore(defaultQueueKey.QueueIndex(item.WorkflowID.String()), item.ID)
	parsed := time.UnixMilli(int64(score))
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Millisecond), parsed, 10*time.Millisecond)
}

func requirePartitionScoreEquals(t *testing.T, r *miniredis.Miniredis, wid uuid.UUID, expected time.Time) {
	t.Helper()
	score, err := r.ZScore(defaultQueueKey.PartitionIndex(), wid.String())
	parsed := time.Unix(int64(score), 0)
	require.NoError(t, err)
	require.WithinDuration(t, expected.Truncate(time.Second), parsed, time.Millisecond)
}
