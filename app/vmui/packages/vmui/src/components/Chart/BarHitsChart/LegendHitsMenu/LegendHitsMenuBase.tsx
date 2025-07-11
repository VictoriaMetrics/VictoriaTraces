import React, { FC } from "preact/compat";
import LegendHitsMenuRow from "./LegendHitsMenuRow";
import useCopyToClipboard from "../../../../hooks/useCopyToClipboard";
import { CopyIcon, FilterIcon, FilterOffIcon } from "../../../Main/Icons";
import { LegendTraceHits, LegendTraceHitsMenu } from "../../../../api/types";
import { TRACES_GROUP_BY } from "../../../../constants/traces";

interface Props {
  legend: LegendTraceHits;
  onApplyFilter: (value: string) => void;
  onClose: () => void;
}

const LegendHitsMenuBase: FC<Props> = ({ legend, onApplyFilter, onClose }) => {
  const copyToClipboard = useCopyToClipboard();

  const handleAddStreamToFilter = () => {
    onApplyFilter(`${TRACES_GROUP_BY}: ${legend.label}`);
    onClose();
  };

  const handleExcludeStreamToFilter = () => {
    onApplyFilter(`(NOT ${TRACES_GROUP_BY}: ${legend.label})`);
    onClose();
  };

  const handlerCopyLabel = async () => {
    await copyToClipboard(legend.label, `${legend.label} has been copied`);
    onClose();
  };

  const options: LegendTraceHitsMenu[] = [
    {
      title: `Copy ${TRACES_GROUP_BY} name`,
      icon: <CopyIcon/>,
      handler: handlerCopyLabel,
    },
    {
      title: `Add ${TRACES_GROUP_BY} to filter`,
      icon: <FilterIcon/>,
      handler: handleAddStreamToFilter,
    },
    {
      title: `Exclude ${TRACES_GROUP_BY} to filter`,
      icon: <FilterOffIcon/>,
      handler: handleExcludeStreamToFilter,
    }
  ];

  return (
    <div className="vm-legend-hits-menu-section">
      {options.map(({ icon, title, handler }) => (
        <LegendHitsMenuRow
          key={title}
          iconStart={icon}
          title={title}
          handler={handler}
        />
      ))}
    </div>
  );
};

export default LegendHitsMenuBase;
