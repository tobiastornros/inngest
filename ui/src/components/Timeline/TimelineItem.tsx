import classNames from '../../utils/classnames'
import statusStyles from '../../utils/statusStyles'

export default function TimelineItem({
  children,
  status,
  active,
  topLine = true,
  bottomLine = true,
}) {
  const itemStatus = statusStyles(status)

  return (
    <li className="flex pr-3.5 relative group items-stretch">
      <div className="basis-[36px] shrink-0 flex flex-col items-center">
        <div
          className={classNames(
            topLine ? 'bg-slate-700' : '',
            `w-[2px] bg-transparent h-[27px] mb-2`
          )}
        ></div>
        <div className="w-full flex items-center justify-center h-[12px]">
          <itemStatus.icon />
        </div>
        {bottomLine && (
          <div
            className={classNames(
              bottomLine ? `bg-slate-700` : ``,
              `w-[2px] bg-transparent mt-2 h-[20px]`
            )}
          ></div>
        )}
      </div>
      <div className="flex items-start min-w-0 w-full">{children}</div>
    </li>
  )
}
