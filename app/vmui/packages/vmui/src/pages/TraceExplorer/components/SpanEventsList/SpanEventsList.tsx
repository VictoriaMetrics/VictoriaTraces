import { useMemo } from "preact/compat";
import cx from "classnames";

import { ErrorIcon } from "../../../../components/Main/Icons";
import { formatDuration } from "../../../../utils/date";
import { SpanEvent } from "../../types";

import "./style.scss";

type SpanEventsListProps = {
  events: ReadonlyArray<SpanEvent>;
  spanStartTime: number;
};

function getAttribute(event: SpanEvent, key: string): string | undefined {
  const attr = event.attributes.find(a => a.key === key);
  return attr ? String(attr.value) : undefined;
}

export default function SpanEventsList({ events, spanStartTime }: SpanEventsListProps) {
  const sortedEvents = useMemo(() => [...events].sort((a, b) => a.timestamp - b.timestamp), [events]);

  return (
    <div className="vm-span-events">
      {sortedEvents.map((event, i) => {
        const offset = event.timestamp - spanStartTime;
        // OTel semantic convention: exceptions are recorded as an event literally named
        // "exception", carrying exception.type/message/stacktrace attributes.
        const isException = event.name === "exception";
        const exceptionMessage = isException ? getAttribute(event, "exception.message") : undefined;

        return (
          <div
            className="vm-span-events__header"
            // eslint-disable-next-line @eslint-react/no-array-index-key -- multiple events can share the same timestamp; index is a stable tie-breaker for the whole-list-replace render
            key={`${event.timestamp}-${i}`}
          >
            {isException && <span className="vm-span-events__error-icon"><ErrorIcon /></span>}
            <span className={cx("vm-span-events__name", { "vm-span-events__name_error": isException })}>
              {event.name}
            </span>
            {exceptionMessage && <span className="vm-span-events__message">{exceptionMessage}</span>}
            <span className="vm-span-events__timing">
              <span
                className="vm-span-events__offset"
                title="Time since span start"
              >
                (+{formatDuration(offset)})
              </span>
            </span>
          </div>
        );
      })}
    </div>
  );
}
