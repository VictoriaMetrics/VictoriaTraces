import { FC } from "preact/compat";
import { LOGSQL_DOCS_URL } from "../../../../../constants/logs";
import Button from "../../../../Main/Button";
import { OpenNewIcon, PlayIcon } from "../../../../Main/Icons";
import Tooltip from "../../../../Main/Tooltip";
import { QueryExample } from "../types";

type Props = {
  example: QueryExample;
  onApply: (value: string) => void;
}

const QueryExamplesItemControls: FC<Props> = ({ example, onApply }) => {
  const url = `${LOGSQL_DOCS_URL}#${example.docAnchor || ""}`;

  const handleApplyQuery = () => {
    onApply(example.query);
  };

  return (
    <div className="vm-query-examples-content-item-header-controls">
      <Button
        as={"a"}
        href={url}
        target="_blank"
        rel="noreferrer"
        startIcon={<OpenNewIcon/>}
        variant="text"
        size="small"
        color="gray"
      >
        Docs
      </Button>

      <Tooltip
        title={"Replace current query and run search"}
        placement="top-right"
      >
        <Button
          variant="text"
          size="small"
          color="primary"
          startIcon={<PlayIcon/>}
          onClick={handleApplyQuery}
        >
          Run
        </Button>
      </Tooltip>
    </div>
  );
};

export default QueryExamplesItemControls;
