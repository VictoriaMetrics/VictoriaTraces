import { FC, useMemo } from "preact/compat";
import Tabs, { TabItemType } from "../../../../components/Main/Tabs/Tabs";
import { TraceChartDefinition } from "../../charts/types";
import "./style.scss";

interface Props {
  charts: TraceChartDefinition[];
  activeId: string;
  onChange: (id: string) => void;
}

const ChartModeToggle: FC<Props> = ({ charts, activeId, onChange }) => {
  const items = useMemo<TabItemType[]>(
    () => charts.map(chart => ({ value: chart.id, label: chart.label })),
    [charts]
  );

  return (
    <div className="vm-chart-mode-toggle">
      <Tabs
        activeItem={activeId}
        items={items}
        onChange={onChange}
      />
    </div>
  );
};

export default ChartModeToggle;
