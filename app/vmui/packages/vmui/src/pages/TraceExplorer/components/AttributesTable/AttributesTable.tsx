import Button from "../../../../components/Main/Button";
import { CopyIcon, LabelIcon } from "../../../../components/Main/Icons";
import Tooltip from "../../../../components/Main/Tooltip";
import useCopyToClipboard from "../../../../hooks/useCopyToClipboard";
import { Attribute } from "../../types";
import "./style.scss";

function extractValue(value: Attribute["value"]): string {
  if (typeof value === "string") return value;
  if (value == null) return "";
  if (value instanceof Uint8Array) return value.toString();
  return JSON.stringify(value);
}

type AttributesTableProps = {
  data: ReadonlyArray<Attribute>;
};

export default function AttributesTable({ data }: AttributesTableProps) {
  const copyToClipboard = useCopyToClipboard();

  return (
    <div className="vm-trace-attributes">
      {data.map((row, i) => {
        const value = extractValue(row.value);
        return (
          <div
            className="vm-trace-attributes__row"
            // eslint-disable-next-line @eslint-react/no-array-index-key -- resource_attr:/span_attr: keys are stripped to the same suffix and can collide; index is a stable tie-breaker for the whole-list-replace render
            key={`${row.key}-${i}`}
          >
            <span className="vm-trace-attributes__key-icon"><LabelIcon /></span>
            <div className="vm-trace-attributes__content">
              <div className="vm-trace-attributes__key">{row.key}</div>
              <div className="vm-trace-attributes__value">{value}</div>
            </div>
            <Tooltip title="Copy value">
              <Button
                className="vm-trace-attributes__copy-icon"
                variant="text"
                size="small"
                style={{ width: "18px", height: "18px", padding: 0 }}
                startIcon={<CopyIcon />}
                aria-label="Copy value"
                onClick={() => copyToClipboard(value, "Copied to clipboard")}
              />
            </Tooltip>
          </div>
        );
      })}
    </div>
  );
}
