import { FC } from "preact/compat";
import LegendHitsMenuRow from "./LegendHitsMenuRow";
import useCopyToClipboard from "../../../../hooks/useCopyToClipboard";
import { CopyIcon, FilterIcon, FilterOffIcon } from "../../../Main/Icons";
import { LegendLogHits, LegendLogHitsMenu } from "../../../../api/types";
import { ApplyHitsFilter } from "../types";

interface Props {
  legend: LegendLogHits;
  onApplyFilter: ApplyHitsFilter;
  onClose: () => void;
}

const LegendHitsMenuBase: FC<Props> = ({ legend, onApplyFilter, onClose }) => {
  const copyToClipboard = useCopyToClipboard();

  const handleAddToFilter = () => {
    onApplyFilter(legend.label, "include");
    onClose();
  };

  const handleExcludeFromFilter = () => {
    onApplyFilter(legend.label, "exclude");
    onClose();
  };

  const handleCopyLabel = async () => {
    await copyToClipboard(legend.label, `${legend.label} has been copied`);
    onClose();
  };

  const options: LegendLogHitsMenu[] = [
    {
      title: "Copy service name",
      iconStart: <CopyIcon/>,
      handler: handleCopyLabel,
    },
    {
      title: "Add service to filter",
      iconStart: <FilterIcon/>,
      handler: handleAddToFilter,
    },
    {
      title: "Exclude service from filter",
      iconStart: <FilterOffIcon/>,
      handler: handleExcludeFromFilter,
    }
  ];

  return (
    <div className="vm-legend-hits-menu-section">
      {options.map(({ iconStart, title, handler }) => (
        <LegendHitsMenuRow
          key={title}
          iconStart={iconStart}
          title={title}
          handler={handler}
        />
      ))}
    </div>
  );
};

export default LegendHitsMenuBase;
