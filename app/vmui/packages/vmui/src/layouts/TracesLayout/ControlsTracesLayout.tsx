import { FC, useRef } from "react";
import classNames from "classnames";
import GlobalSettings, { ChildComponentHandle } from "../../components/Configurators/GlobalSettings/GlobalSettings";
import { ControlsProps } from "../Header/HeaderControls/HeaderControls";
import { TimeSelector } from "../../components/Configurators/TimeRangeSettings/TimeSelector/TimeSelector";
import TenantsFields from "../../components/Configurators/GlobalSettings/TenantsConfiguration/TenantsFields";
import { ExecutionControls } from "../../components/Configurators/TimeRangeSettings/ExecutionControls/ExecutionControls";
import DurationRangeFilter from "../../components/Configurators/DurationRangeFilter/DurationRangeFilter";
import TracesLimitConfigurator from "../../components/Configurators/GlobalSettings/TracesLimitConfigurator/TracesLimitConfigurator";

const ControlsLogsLayout: FC<ControlsProps> = ({ isMobile }) => {
  const tracesLimitRef = useRef<ChildComponentHandle>(null);

  return (
    <div
      className={classNames({
        "vm-header-controls": true,
        "vm-header-controls_mobile": isMobile,
      })}
    >
      <DurationRangeFilter/>
      <TenantsFields/>
      <TimeSelector/>
      <ExecutionControls/>
      <GlobalSettings
        extraControls={[{ show: true, component: <TracesLimitConfigurator ref={tracesLimitRef}/> }]}
        onApplyExtra={() => tracesLimitRef.current?.handleApply()}
      />
    </div>
  );
};

export default ControlsLogsLayout;
