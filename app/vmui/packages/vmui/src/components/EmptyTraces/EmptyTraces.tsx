import { FC } from "preact/compat";
import "./style.scss";

const EmptyTraces: FC = () => {
  return (
    <div className="vm-empty">No traces found</div>
  );
};

export default EmptyTraces;
