import classNames from '../../utils/classnames'
import statusStyles from '../../utils/statusStyles'

export default function TimelineRow({
  status,
  children,
  topLine = true,
  bottomLine = true,
  iconOffset = 0,
}) {
  const itemStatus = statusStyles(status)

  return (
    <li className="flex">
      <div className="flex flex-col items-center basis-[36px]">
        <div
          className={classNames(
            bottomLine ? `bg-slate-700` : ``,
            `w-[2px] shrink-0 bg-transparent`
          )}
          style={`flex-basis: ${iconOffset}px`}
        ></div>
        <div className="basis-[24px] shrink-0 flex items-center">
          <itemStatus.icon />
        </div>
        <div
          className={classNames(
            bottomLine ? `bg-slate-700` : ``,
            `basis-[100%] w-[2px] bg-transparent`
          )}
        ></div>
      </div>
      <div className="w-full pb-4 min-w-0">{children}</div>
    </li>
  )
}
