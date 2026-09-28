import { FC, useCallback, useEffect, useMemo, useRef } from "preact/compat";
import { CSSProperties } from "preact";
import classNames from "classnames";
import Accordion from "../../../../components/Main/Accordion";
import Select from "../../../../components/Main/Select";
import TextField from "../../../../components/Main/TextField";
import Button from "../../../../components/Main/Button";
import Tooltip from "../../../../components/Main/Tooltip";
import { CloseIcon, SpinnerIcon, WidthIcon } from "../../../../components/Main/Icons";
import DragResizeHandle from "../../../../components/Main/DragResizeHandle";
import CheckboxListFilter from "./CheckboxListFilter";
import { DurationRequest, useFiltersSidebarState } from "../../hooks/useFiltersSidebarState";
import { useFiltersSidebarWidth } from "../../hooks/useFiltersSidebarWidth";
import { useAvailableHeight } from "../../hooks/useAvailableHeight";
import { useFieldValues } from "../../hooks/useFieldValues";
import { useFieldNames } from "../../hooks/useFieldNames";
import { useExtraFilters } from "../../hooks/useExtraFilters";
import { useAppState } from "../../../../state/common/StateContext";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";
import colorGenerator from "../../../../utils/color-generator";
import {
  DURATION_PLACEHOLDER,
  groupQueryField,
  mapCodesToSpanKinds,
  mapSpanKindsToCodes,
  OPERATION_FIELD,
  SERVICE_FIELD,
  SPAN_KIND_OPTIONS,
  validateDurationInput,
} from "../../utils";
import "./style.scss";

const KIND_FIELD = "kind";

interface Props {
  query: string;
  onClose: () => void;
  durationRequest?: DurationRequest | null;
}

const SectionTitle: FC<{
  title: string,
  count: number,
  total?: number,
  loading?: boolean,
  emptyLabel?: string,
  valueLabel?: string,
}> = ({
  title,
  count,
  total,
  loading,
  emptyLabel,
  valueLabel,
}) => {
  const label = count > 0
    ? (valueLabel ?? `${count}${total !== undefined ? `/${total}` : ""}`)
    : (total !== undefined ? total : emptyLabel);
  return (
    <span className="vm-filters-sidebar-section__title">
      {title}
      {loading ? (
        <span className="vm-filters-sidebar-section__spinner">
          <SpinnerIcon/>
        </span>
      ) : (
        !!label && <span className="vm-filters-sidebar-section__badge">({label})</span>
      )}
    </span>
  );
};

const FiltersSidebar: FC<Props> = ({ query, onClose, durationRequest }) => {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const { width, setWidth, clearWidth } = useFiltersSidebarWidth();
  const { height, top } = useAvailableHeight(sidebarRef);
  const { isDarkTheme: isDarkThemeState } = useAppState();
  const isDarkTheme = isDarkThemeState ?? true;
  const { isMobile } = useDeviceDetect();

  const sidebarStyles: CSSProperties | undefined = useMemo(() => {
    if (isMobile) return;

    const styles: CSSProperties = { top };
    if (width) styles.width = width;
    if (height) styles.height = height;
    return styles;
  }, [height, top, width, isMobile]);

  useEffect(() => {
    if (!isMobile) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  }, [isMobile]);

  const {
    minDuration,
    setMinDuration,
    maxDuration,
    setMaxDuration,
    tagName,
    setTagName,
    tagValue,
    setTagValue,
  } = useFiltersSidebarState(durationRequest);

  const { extraFilters, toggleFilter, addFilter, removeFilter, selectedValues } = useExtraFilters();
  const selectedServices = useMemo(() => selectedValues(SERVICE_FIELD), [selectedValues]);
  const selectedOperations = useMemo(() => selectedValues(OPERATION_FIELD), [selectedValues]);
  const selectedKinds = useMemo(
    () => mapCodesToSpanKinds(selectedValues(KIND_FIELD)),
    [selectedValues]
  );
  const tagFilters = useMemo(
    () => extraFilters.filter(f => ![SERVICE_FIELD, OPERATION_FIELD, KIND_FIELD].includes(f.field)),
    [extraFilters]
  );

  const toggleService = useCallback(
    (value: string) => toggleFilter(SERVICE_FIELD, value),
    [toggleFilter]
  );
  const toggleOperation = useCallback(
    (value: string) => toggleFilter(OPERATION_FIELD, value),
    [toggleFilter]
  );
  const toggleKind = useCallback(
    (value: string) => toggleFilter(KIND_FIELD, mapSpanKindsToCodes([value])[0]),
    [toggleFilter]
  );

  const { values: serviceValues, loading: serviceLoading } = useFieldValues(SERVICE_FIELD, query);
  const { values: operationValues, loading: operationLoading } = useFieldValues(OPERATION_FIELD, query);
  const { groups: tagNameGroups } = useFieldNames(query);
  const tagNames = useMemo(() => tagNameGroups.map(g => g.displayName), [tagNameGroups]);
  const tagNameMeta = useMemo(
    () => tagNameGroups.map(g => (g.isIndexed ? "array" : "")),
    [tagNameGroups]
  );
  const selectedTagGroup = useMemo(
    () => tagNameGroups.find(g => g.displayName === tagName),
    [tagNameGroups, tagName]
  );
  const { values: tagValues } = useFieldValues(selectedTagGroup?.realNames[0] ?? tagName, query);

  const minDurationError = validateDurationInput(minDuration);
  const maxDurationError = validateDurationInput(maxDuration);

  return (
    <div
      className={classNames({
        "vm-filters-sidebar": true,
        "vm-filters-sidebar_mobile": isMobile,
      })}
      ref={sidebarRef}
      style={sidebarStyles}
    >
      <div className="vm-filters-sidebar-header">
        <span className="vm-filters-sidebar-header__title">Filters</span>
        <div className="vm-filters-sidebar-header__actions">
          <Tooltip title="Reset width">
            <Button
              variant="text"
              color="gray"
              size="small"
              onClick={clearWidth}
              startIcon={<WidthIcon/>}
            />
          </Tooltip>
          <Tooltip title="Hide filters">
            <Button
              variant="text"
              color="gray"
              size="small"
              onClick={onClose}
              startIcon={<CloseIcon/>}
            />
          </Tooltip>
        </div>
      </div>
      <div className="vm-filters-sidebar-content">
        <div className="vm-filters-sidebar-section">
          <Accordion
            title={<SectionTitle
              title="Service"
              count={selectedServices.length}
              total={serviceValues.length}
              loading={serviceLoading}
            />}
          >
            <CheckboxListFilter
              values={serviceValues}
              selected={selectedServices}
              loading={serviceLoading}
              onToggle={toggleService}
              emptyText="No services found"
              renderIcon={service => (
                <span
                  className="vm-filters-checkbox-list__item-dot"
                  style={{ backgroundColor: colorGenerator.getColorByKey(service, isDarkTheme) }}
                />
              )}
            />
          </Accordion>
        </div>

        <div className="vm-filters-sidebar-section">
          <Accordion
            title={<SectionTitle
              title="Operation"
              count={selectedOperations.length}
              total={operationValues.length}
              loading={operationLoading}
            />}
          >
            {/* TODO: scope operations to selectedServices once cross-filtering is needed */}
            <CheckboxListFilter
              values={operationValues}
              selected={selectedOperations}
              loading={operationLoading}
              onToggle={toggleOperation}
              emptyText="No operations found"
            />
          </Accordion>
        </div>

        <div className="vm-filters-sidebar-section">
          <Accordion
            title={<SectionTitle
              title="Duration"
              count={(minDuration ? 1 : 0) + (maxDuration ? 1 : 0)}
              emptyLabel="any"
              valueLabel={
                minDuration && maxDuration
                  ? `${minDuration}–${maxDuration}`
                  : minDuration
                    ? `≥${minDuration}`
                    : `≤${maxDuration}`
              }
            />}
          >
            <div className="vm-filters-sidebar-duration">
              <TextField
                value={minDuration}
                error={minDurationError}
                placeholder={DURATION_PLACEHOLDER}
                onChange={setMinDuration}
              />
              <span className="vm-filters-sidebar-duration__separator">–</span>
              <TextField
                value={maxDuration}
                error={maxDurationError}
                placeholder={DURATION_PLACEHOLDER}
                onChange={setMaxDuration}
              />
            </div>
          </Accordion>
        </div>

        <div className="vm-filters-sidebar-section">
          <Accordion
            title={<SectionTitle
              title="Span type"
              count={selectedKinds.length}
              total={SPAN_KIND_OPTIONS.length}
            />}
          >
            <CheckboxListFilter
              values={SPAN_KIND_OPTIONS}
              selected={selectedKinds}
              onToggle={toggleKind}
            />
          </Accordion>
        </div>

        <div className="vm-filters-sidebar-section">
          <Accordion
            title={<SectionTitle
              title="Tags"
              count={tagFilters.length}
            />}
          >
            <div className="vm-filters-sidebar-tags">
              <div className="vm-filters-sidebar-tags-selects">
                <Select
                  placeholder="Name"
                  value={tagName}
                  list={tagNames}
                  metaList={tagNameMeta}
                  searchable
                  clearable
                  onChange={setTagName}
                />
                <Select
                  placeholder="Value"
                  value={tagValue}
                  list={tagValues}
                  searchable
                  clearable
                  disabled={!tagName}
                  onChange={setTagValue}
                />
                <Button
                  variant="outlined"
                  fullWidth
                  disabled={!tagName || !tagValue}
                  onClick={() => {
                    const field = selectedTagGroup ? groupQueryField(selectedTagGroup) : tagName;
                    if (!field) return;
                    addFilter(field, tagValue);
                    setTagName("");
                  }}
                >
                  Add tag
                </Button>
              </div>
              {!!tagFilters.length && (
                <div className="vm-filters-sidebar-tags-badges">
                  {tagFilters.map(tag => (
                    <div
                      key={`${tag.field}:${tag.value}`}
                      className="vm-filters-sidebar-tags-badge"
                    >
                      <span className="vm-filters-sidebar-tags-badge__label">
                        {tag.field}: {tag.value}
                      </span>
                      <div
                        className="vm-filters-sidebar-tags-badge__remove"
                        onClick={() => removeFilter(tag.field, tag.value)}
                      >
                        <CloseIcon/>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Accordion>
        </div>
      </div>
      <DragResizeHandle
        targetRef={sidebarRef}
        minSize={220}
        dir={1}
        onResizeEnd={setWidth}
      />
    </div>
  );
};

export default FiltersSidebar;
