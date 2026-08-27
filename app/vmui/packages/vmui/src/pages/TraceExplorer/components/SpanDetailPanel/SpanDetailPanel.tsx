import { RefObject, useMemo, useState } from "preact/compat";
import { useNavigate } from "react-router-dom";

import Tabs from "../../../../components/Main/Tabs";
import TextField from "../../../../components/Main/TextField";
import Select from "../../../../components/Main/Select";
import Switch from "../../../../components/Main/Switch";
import Button from "../../../../components/Main/Button";
import Tooltip from "../../../../components/Main/Tooltip";
import { CloseIcon, SearchIcon, OpenNewIcon } from "../../../../components/Main/Icons";
import AttributesTable from "../AttributesTable";
import TimeScale from "../TimeScale";
import { formatDuration } from "../../../../utils/date";
import colorGenerator from "../../../../utils/color-generator";
import { useAppState } from "../../../../state/common/StateContext";
import { Attribute, SpanEvent, Link, OtelSpan } from "../../types";

import "./style.scss";

type SpanDetailPanelProps = {
  span: OtelSpan;
  panelRef: RefObject<HTMLDivElement>;
  width: number;
  onClose: () => void;
};

function filterAttributes(attributes: ReadonlyArray<Attribute>, query: string): ReadonlyArray<Attribute> {
  if (!query) return attributes;
  const q = query.toLowerCase();
  return attributes.filter(({ key, value }) => key.toLowerCase().includes(q) || String(value).toLowerCase().includes(q));
}

const SpanDetailPanel = ({ span, panelRef, width, onClose }: SpanDetailPanelProps) => {
  const [activeTab, setActiveTab] = useState("fields");
  const [search, setSearch] = useState("");
  const [selectedEvent, setSelectedEvent] = useState<SpanEvent | null>(null);
  const [showAbsoluteTime, setShowAbsoluteTime] = useState(false);
  const [prevSpanID, setPrevSpanID] = useState(span.spanID);
  const navigate = useNavigate();

  // Selecting a different span in the waterfall re-renders this same panel instance rather
  // than remounting it, so the previous span's event selection would otherwise stick around.
  if (prevSpanID !== span.spanID) {
    setPrevSpanID(span.spanID);
    setSelectedEvent(null);
  }

  const intrinsicFields: Attribute[] = useMemo(() => [
    { key: "Trace ID", value: span.traceID },
    { key: "Service", value: span.resource.serviceName },
    { key: "Operation", value: span.name },
    { key: "Duration", value: formatDuration(span.duration) },
  ], [span]);

  const { isDarkTheme } = useAppState();
  const color = colorGenerator.getColorByKey(span.resource.serviceName, isDarkTheme ?? true);

  const allFields = useMemo(() => [...intrinsicFields, ...span.attributes], [intrinsicFields, span.attributes]);
  const filteredFields = useMemo(() => filterAttributes(allFields, search), [allFields, search]);

  const sortedEvents = useMemo(
    () => [...span.events].sort((a, b) => a.timestamp - b.timestamp),
    [span.events]
  );
  const eventNames = useMemo(() => sortedEvents.map(event => event.name), [sortedEvents]);
  const eventOffsets = useMemo(
    () => sortedEvents.map(event => `+${formatDuration(event.timestamp - span.startTime)}`),
    [sortedEvents, span.startTime]
  );
  const selectEventByName = (name: string) => {
    setSelectedEvent(sortedEvents.find(event => event.name === name) ?? null);
  };

  const openLinkedTrace = (link: Link) => {
    navigate(
      `/trace?trace_id=${encodeURIComponent(link.traceID)}&span_id=${encodeURIComponent(link.spanID)}`,
      { state: { autoRun: true } }
    );
  };

  return (
    <div
      className="vm-span-detail-panel"
      ref={panelRef}
      style={width ? { width: `${width}px` } : undefined}
    >
      <div className="vm-span-detail-panel__header">
        <span className="vm-span-detail-panel__header-label">Span ID:</span>
        <span className="vm-span-detail-panel__header-id">{span.spanID}</span>
        <Tooltip title="Close">
          <Button
            variant="text"
            color="gray"
            size="small"
            onClick={onClose}
            startIcon={<CloseIcon/>}
          />
        </Tooltip>
      </div>

      <div className="vm-span-detail-panel__content">
        <Tabs
          activeItem={activeTab}
          onChange={setActiveTab}
          items={[
            { value: "fields", label: `Fields (${span.attributes.length})` },
            { value: "events", label: `Events (${span.events.length})` },
            { value: "links", label: `Links (${span.links.length})` },
          ]}
        />

        {activeTab === "fields" && (
          <>
            <TextField
              placeholder="Filter fields"
              startIcon={<SearchIcon />}
              value={search}
              onChange={setSearch}
              type="search"
            />
            <AttributesTable data={filteredFields} />
          </>
        )}

        {activeTab === "events" && (
          span.events.length === 0 ? (
            <div className="vm-span-detail-panel__empty">No events on this span.</div>
          ) : (
            <>
              <div className="vm-span-detail-panel__events-header">
                <Switch
                  value={showAbsoluteTime}
                  onChange={setShowAbsoluteTime}
                  label="Absolute time"
                />
              </div>
              <div className="vm-span-detail-panel__events-timeline">
                <TimeScale
                  mode="events"
                  numTicks={4}
                  color={color}
                  startTime={span.startTime}
                  endTime={span.endTime}
                  events={span.events}
                  selectedEvent={selectedEvent}
                  onSelectEvent={setSelectedEvent}
                  absolute={showAbsoluteTime}
                />
              </div>
              <Select
                label="Event"
                placeholder="Select event"
                value={selectedEvent?.name ?? ""}
                list={eventNames}
                metaList={eventOffsets}
                searchable
                clearable
                onChange={selectEventByName}
              />
              {selectedEvent && (
                selectedEvent.attributes.length > 0 ? (
                  <AttributesTable data={selectedEvent.attributes} />
                ) : (
                  <div className="vm-span-detail-panel__empty">No attributes on this event.</div>
                )
              )}
            </>
          )
        )}

        {activeTab === "links" && (
          <>
            {span.links.length === 0 && (
              <div className="vm-span-detail-panel__empty">No links on this span.</div>
            )}
            {span.links.map((link, i) => (
              <div
                className="vm-span-detail-panel__event"
                key={`${link.traceID}-${link.spanID}`}
              >
                <div className="vm-span-detail-panel__link-header">
                  <span className="vm-span-detail-panel__link-header-label">Link #{i + 1}</span>
                  <Tooltip title="Open linked trace">
                    <Button
                      variant="text"
                      color="gray"
                      size="small"
                      startIcon={<OpenNewIcon />}
                      aria-label="Open linked trace"
                      onClick={() => openLinkedTrace(link)}
                    />
                  </Tooltip>
                </div>
                <AttributesTable
                  data={[
                    { key: "Trace ID", value: link.traceID },
                    { key: "Span ID", value: link.spanID },
                    ...link.attributes,
                  ]}
                />
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  );
};

export default SpanDetailPanel;
