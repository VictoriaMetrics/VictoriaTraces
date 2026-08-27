import { FC, useRef } from "preact/compat";
import { PlayIcon, SpinnerIcon } from "../../../../components/Main/Icons";
import "./style.scss";
import Button from "../../../../components/Main/Button";
import Tooltip from "../../../../components/Main/Tooltip";
import TextField from "../../../../components/Main/TextField";
import TracesLimitInput from "../../../../components/Configurators/TracesLimitController/TracesLimitInput";
import QueryEditor from "../../../../components/Configurators/QueryEditor";
import LogsQueryEditorAutocomplete from
  "../../../../components/Configurators/QueryEditor/LogsQL/LogsQueryEditorAutocomplete";
import AutocompleteToggle from "../../../../components/Configurators/QueryEditor/AutocompleteToggle";
import QueryHistory from "../../../../components/QueryHistory";
import { getHistoryFromStorage } from "../../../../components/QueryHistory/utils";
import QueryExamplesButton from "../../../../components/Configurators/QueryEditor/QueryExamples/QueryExamplesButton";
import { useQueryState } from "../../../../state/query/QueryStateContext";
import { useQuickAutocomplete } from "../../../../hooks/useQuickAutocomplete";
import FiltersSidebarToggle from "../FiltersSidebar/FiltersSidebarToggle";

export type TraceQueryMode = "search" | "traceId";

interface Props {
  mode: TraceQueryMode;
  traceId: string;
  onChangeTraceId: (val: string) => void;
  query: string;
  onChangeQuery: (val: string) => void;
  limit?: number;
  onChangeLimit?: (val: number) => void;
  isLoading: boolean;
  onRun: (query?: string) => void;
}

const TraceExplorerHeader: FC<Props> = ({
  mode,
  traceId,
  onChangeTraceId,
  query,
  onChangeQuery,
  limit,
  onChangeLimit,
  isLoading,
  onRun,
}) => {
  const { autocompleteQuick } = useQueryState();
  const setQuickAutocomplete = useQuickAutocomplete();

  const historyIndexRef = useRef<number | null>(null);

  const handleHistoryChange = (step: number) => {
    const history = getHistoryFromStorage();
    if (!history.length) return;

    const currentIndex = historyIndexRef.current ?? (step > 0 ? -1 : 0);
    const nextIndex = currentIndex + step;

    if (nextIndex < 0 || nextIndex >= history.length) return;

    historyIndexRef.current = nextIndex;
    onChangeQuery(history[nextIndex].query);
  };

  const createHandlerArrow = (step: number) => () => {
    handleHistoryChange(step);
  };

  const handleApplyQuery = (value: string) => {
    onChangeQuery(value);
    onRun(value);
  };

  const onChangeHandle = (value: string) => {
    historyIndexRef.current = null;
    onChangeQuery(value);

    if (autocompleteQuick) {
      setQuickAutocomplete(false);
    }
  };

  const executeButton = (
    <Button
      className="vm-trace-explorer-header-bottom-execute"
      startIcon={isLoading ? <SpinnerIcon/> : <PlayIcon/>}
      onClick={() => onRun()}
      fullWidth
    >
      <div>
        <span className="vm-trace-explorer-header-bottom-execute__text">
          {isLoading ? "Cancel" : "Execute"}
        </span>
        <span className="vm-trace-explorer-header-bottom-execute__text_hidden">Execute</span>
      </div>
    </Button>
  );

  const executeIconButton = (
    <Tooltip title={isLoading ? "Cancel" : "Execute"}>
      <div className="vm-trace-explorer-header-top-traceid__button">
        <Button
          variant="text"
          color="primary"
          startIcon={isLoading ? <SpinnerIcon/> : <PlayIcon/>}
          onClick={() => onRun()}
          aria-label={isLoading ? "Cancel" : "Execute"}
        />
      </div>
    </Tooltip>
  );

  return (
    <>
      {mode === "traceId" ? (
        <div className="vm-trace-explorer-header-top-traceid">
          <div className="vm-trace-explorer-header-top-traceid__input">
            <TextField
              label="TraceID"
              value={traceId}
              placeholder="Paste a trace ID"
              onChange={onChangeTraceId}
              onEnter={onRun}
            />
          </div>
          {executeIconButton}
        </div>
      ) : (
        <div className="vm-trace-explorer-header-top">
          <div className="vm-trace-explorer-header-top-fields">
            <QueryEditor
              label="Search"
              value={query}
              onChange={onChangeHandle}
              onEnter={onRun}
              onArrowUp={createHandlerArrow(1)}
              onArrowDown={createHandlerArrow(-1)}
              autocomplete
              autocompleteEl={LogsQueryEditorAutocomplete}
              includeFunctions={false}
            />
          </div>
          <TracesLimitInput
            limit={limit || 0}
            onChangeLimit={onChangeLimit || (() => {})}
            onPressEnter={onRun}
          />
        </div>
      )}
      {mode === "search" && (
        <div className="vm-trace-explorer-header-bottom">
          <div className="vm-trace-explorer-header-bottom__toggle">
            <FiltersSidebarToggle/>
          </div>
          <QueryExamplesButton onApply={handleApplyQuery}/>
          <AutocompleteToggle/>
          <QueryHistory handleSelectQuery={handleApplyQuery}/>
          {executeButton}
        </div>
      )}
    </>
  );
};

export default TraceExplorerHeader;
