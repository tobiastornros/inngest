import Button from '../Button'
import { useState } from 'preact/hooks'
import classNames from '../../utils/classnames'

export default function CodeBlock({ tabs }) {
  const [activeTab, setActiveTab] = useState(0)

  const handleTabClick = (index) => {
    setActiveTab(index)
  }

  const handleCopyClick = () => {
    navigator.clipboard.writeText(tabs[activeTab].content)
  }

  return (
    <div className="w-full bg-slate-800/30 border border-slate-700/30 rounded-lg shadow">
      <div className="bg-slate-800/40 flex justify-between shadow">
        <div className="flex">
          {tabs.map((tab, i) => (
            <button
              className={classNames(
                i === activeTab
                  ? `border-indigo-400 text-white`
                  : `border-transparent text-slate-400`,
                `text-xs px-5 py-2.5 border-b-2 block transition-all duration-150`
              )}
              onClick={() => handleTabClick(i)}
              key={i}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="flex gap-2 items-center mr-2">
          <Button label="Copy" btnAction={handleCopyClick} />
          <Button label="Expand" />
        </div>
      </div>
      <div className="overflow-hidden grid">
        {tabs.map((tab, i) => (
          <code
            className={classNames(
              i === activeTab ? ` ` : `opacity-0`,
              `col-start-1 row-start-1 transition-all`
            )}
          >
            <pre className="p-4 overflow-x-scroll text-2xs">
              {tabs[i].content}
            </pre>
          </code>
        ))}
      </div>
    </div>
  )
}
