import React, { FC } from "preact/compat";
import GroupTraces from "../../../GroupTraces/GroupTraces";
import { ViewProps } from "../../types";
import EmptyTraces from "../components/EmptyTraces/EmptyTraces";

const MemoizedGroupTraces = React.memo(GroupTraces);

const GroupView: FC<ViewProps> = ({ data, settingsRef }) => {
  if (!data.length) return <EmptyTraces />;

  return (
    <>
      <MemoizedGroupTraces
        traces={data}
        settingsRef={settingsRef}
      />
    </>
  );
};

export default GroupView; 
