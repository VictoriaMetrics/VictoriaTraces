import { FC } from "preact/compat";
import "./style.scss";
import { LegendLogHits } from "../../../../api/types";
import LegendHitsMenuStats from "./LegendHitsMenuStats";
import LegendHitsMenuBase from "./LegendHitsMenuBase";
import LegendHitsMenuRow from "./LegendHitsMenuRow";
import { HITS_FIELDS_LIMIT } from "../../../../constants/logs";
import { ApplyHitsFilter } from "../types";

const otherDescription = `aggregated results for services not in the top ${HITS_FIELDS_LIMIT}`;

interface Props {
  legend: LegendLogHits;
  onApplyFilter: ApplyHitsFilter;
  onClose: () => void;
}

const LegendHitsMenu: FC<Props> = ({ legend, onApplyFilter, onClose }) => {
  return (
    <div className="vm-legend-hits-menu">
      <div className="vm-legend-hits-menu-section">
        <LegendHitsMenuRow
          className="vm-legend-hits-menu-row_info"
          title={legend.isOther ? otherDescription : legend.label}
        />
      </div>

      {!legend.isOther && (
        <LegendHitsMenuBase
          legend={legend}
          onApplyFilter={onApplyFilter}
          onClose={onClose}
        />
      )}

      <LegendHitsMenuStats legend={legend}/>
    </div>
  );
};

export default LegendHitsMenu;
