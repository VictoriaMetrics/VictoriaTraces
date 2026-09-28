import { createContext, FC, useCallback, useContext, useEffect, useRef, useState, ReactNode } from "preact/compat";
import Alert from "../components/Main/Alert";
import useDeviceDetect from "../hooks/useDeviceDetect";
import classNames from "classnames";
import { CloseIcon } from "../components/Main/Icons";

interface SnackbarItem {
  text: string | ReactNode,
  type: "success" | "error" | "info" | "warning"
  timeout?: number
}

export interface SnackModel extends SnackbarItem {
  open?: boolean;
  key?: number;
}

type SnackbarContextType = {
  showInfoMessage: (item: SnackbarItem) => void
};

export const SnackbarContext = createContext<SnackbarContextType>({
  showInfoMessage: () => {
    // default value here makes no sense
  }
});

// eslint-disable-next-line @eslint-react/no-use-context -- preact/compat does not export a 'use' hook, useContext is required here
export const useSnack = (): SnackbarContextType => useContext(SnackbarContext);

export const SnackbarProvider: FC = ({ children }) => {
  const { isMobile } = useDeviceDetect();

  const [snack, setSnack] = useState<SnackModel>({ text: "", type: "info" });
  const [open, setOpen] = useState(false);

  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();

  const handleClose = useCallback(() => {
    clearTimeout(timeoutRef.current);
    setOpen(false);
  }, []);

  const showInfoMessage = useCallback((item: SnackbarItem) => {
    clearTimeout(timeoutRef.current);
    setSnack({
      ...item,
      key: Date.now()
    });
    setOpen(true);
    timeoutRef.current = setTimeout(handleClose, item.timeout || 4000);
  }, [handleClose]);

  useEffect(() => () => clearTimeout(timeoutRef.current), []);

  return <SnackbarContext value={{ showInfoMessage }}>
    {open && <div
      className={classNames({
        "vm-snackbar": true,
        "vm-snackbar_mobile": isMobile,
      })}
    >
      <Alert variant={snack.type}>
        <div className="vm-snackbar-content">
          <span>{snack.text}</span>
          <div
            className="vm-snackbar-content__close"
            onClick={handleClose}
          >
            <CloseIcon/>
          </div>
        </div>
      </Alert>
    </div>}
    {children}
  </SnackbarContext>;
};


