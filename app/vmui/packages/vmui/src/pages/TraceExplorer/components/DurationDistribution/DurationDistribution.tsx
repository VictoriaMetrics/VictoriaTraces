import classNames from "classnames";
import Tooltip from "../../../../components/Main/Tooltip";
import { formatDurationCompact } from "../../../../utils/date";
import { Microseconds } from "../../types";
import { DurationDistribution as DurationDistributionData, findDurationBucketIndex } from "../../hooks/useDurationDistribution";
import "./style.scss";

interface Props {
  distribution: DurationDistributionData;
  operation: string;
  currentDurationNs: number;
}

export default function DurationDistribution({ distribution, operation, currentDurationNs }: Props) {
  const { total, minDurationNs, maxDurationNs, minDurationUs, maxDurationUs, bucketWidthUs, bucketCounts } = distribution;
  const maxBucketCount = Math.max(1, ...bucketCounts);
  const currentBucket = findDurationBucketIndex(currentDurationNs, minDurationNs, maxDurationNs);

  return (
    <div className="vm-trace-info-drawer-duration">
      <div className="vm-trace-info-drawer-duration__caption">
        This trace against {total} trace{total === 1 ? "" : "s"} of{" "}
        <strong>{operation}</strong> in the selected time range.
      </div>
      <div className="vm-trace-info-drawer-duration__bars">
        {bucketCounts.map((count, i) => {
          const lowUs = minDurationUs + i * bucketWidthUs;
          const highUs = minDurationUs + (i + 1) * bucketWidthUs;
          return (
            <Tooltip
              // eslint-disable-next-line @eslint-react/no-array-index-key -- bucketCounts is a fixed-length, positionally-meaningful array (one entry per equal-width duration bucket); index is the only stable identity
              key={i}
              title={
                <div className="vm-trace-info-drawer-duration__tooltip">
                  <div className="vm-trace-info-drawer-duration__tooltip-row">
                    <span className="vm-trace-info-drawer-duration__tooltip-label">Duration</span>
                    <span className="vm-trace-info-drawer-duration__tooltip-value">
                      {formatDurationCompact(lowUs as Microseconds)} – {formatDurationCompact(highUs as Microseconds)}
                    </span>
                  </div>
                  <div className="vm-trace-info-drawer-duration__tooltip-row">
                    <span className="vm-trace-info-drawer-duration__tooltip-label">Traces</span>
                    <span className="vm-trace-info-drawer-duration__tooltip-value">{count}</span>
                  </div>
                </div>
              }
            >
              <span
                className={classNames("vm-trace-info-drawer-duration__bar", {
                  "vm-trace-info-drawer-duration__bar_current": i === currentBucket,
                })}
                style={{ "--bar-intensity": count > 0 ? Math.max(0.15, count / maxBucketCount) : 0 }}
              />
            </Tooltip>
          );
        })}
      </div>
      <div className="vm-trace-info-drawer-duration__range">
        <span>{formatDurationCompact(minDurationUs as Microseconds)}</span>
        <span>{formatDurationCompact(maxDurationUs as Microseconds)}</span>
      </div>
    </div>
  );
}
