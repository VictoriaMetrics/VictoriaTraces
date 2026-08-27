import { FC } from "preact/compat";
import Alert from "../../../../components/Main/Alert";
import { formatApiError } from "../../utils";
import { ApiError } from "../../types";

import "./style.scss";

interface Props {
  error: ApiError;
  className?: string;
}

const ApiErrorAlert: FC<Props> = ({ error, className }) => {
  const { title, rows } = formatApiError(error);

  return (
    <div className={className}>
      <Alert
        variant="error"
        title={title}
      >
        {rows.length > 0 && (
          <table className="vm-api-error-alert-details">
            <tbody>
              {rows.map(row => (
                <tr key={row.name}>
                  <td className="vm-api-error-alert-details__attr">{row.name}</td>
                  <td className="vm-api-error-alert-details__value">{row.value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Alert>
    </div>
  );
};

export default ApiErrorAlert;
