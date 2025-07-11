import { Traces } from "../../../api/types";

export interface ViewProps {
  data: Traces[];
  settingsRef: React.RefObject<HTMLDivElement>;
}
