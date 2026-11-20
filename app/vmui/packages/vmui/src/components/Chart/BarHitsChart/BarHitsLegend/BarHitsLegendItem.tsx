import { FC, useRef, useState, MouseEvent } from "preact/compat";
import classNames from "classnames";
import { LegendLogHits } from "../../../../api/types";
import { formatNumberShort } from "../../../../utils/number";
import Popper from "../../../Main/Popper/Popper";
import useBoolean from "../../../../hooks/useBoolean";
import LegendHitsMenu from "../LegendHitsMenu/LegendHitsMenu";
import { ApplyHitsFilter } from "../types";

interface Props {
  legend: LegendLogHits;
  isHidden: boolean;
  onToggleSeries: (label: string, isMultiSelect: boolean) => void;
  onApplyFilter: ApplyHitsFilter;
}

const BarHitsLegendItem: FC<Props> = ({ legend, isHidden, onToggleSeries, onApplyFilter }) => {
  const {
    value: openContextMenu,
    setTrue: handleOpenContextMenu,
    setFalse: handleCloseContextMenu,
  } = useBoolean(false);

  const legendRef = useRef<HTMLDivElement>(null);
  const [clickPosition, setClickPosition] = useState<{ top: number; left: number } | null>(null);

  const totalShortFormatted = formatNumberShort(legend.total);

  const handleClickByStream = (e: MouseEvent<HTMLDivElement>) => {
    onToggleSeries(legend.label, e.metaKey || e.ctrlKey);
  };

  const handleContextMenu = (e: MouseEvent<HTMLDivElement>) => {
    e.preventDefault();
    setClickPosition({ top: e.clientY, left: e.clientX });
    handleOpenContextMenu();
  };

  return (
    <div
      ref={legendRef}
      className={classNames({
        "vm-bar-hits-legend-item": true,
        "vm-bar-hits-legend-item_other": legend.isOther,
        "vm-bar-hits-legend-item_hide": isHidden,
      })}
      onClick={handleClickByStream}
      onContextMenu={handleContextMenu}
    >
      <div
        className="vm-bar-hits-legend-item__marker"
        style={{ backgroundColor: `${legend.stroke}` }}
      />
      <div className="vm-bar-hits-legend-item__label">{legend.label}</div>
      <span className="vm-bar-hits-legend-item__total">({totalShortFormatted})</span>
      <Popper
        placement="fixed"
        open={openContextMenu}
        buttonRef={legendRef}
        placementPosition={clickPosition}
        onClose={handleCloseContextMenu}
      >
        <LegendHitsMenu
          legend={legend}
          onApplyFilter={onApplyFilter}
          onClose={handleCloseContextMenu}
        />
      </Popper>
    </div>
  );
};

export default BarHitsLegendItem;
