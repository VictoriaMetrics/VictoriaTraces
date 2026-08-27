import { useEffect, useState } from "preact/compat";
import { getFromStorage, removeFromStorage, saveToStorage } from "../../../utils/storage";
import useEventListener from "../../../hooks/useEventListener";
import useDeviceDetect from "../../../hooks/useDeviceDetect";
import { toPrefixedKey } from "../../../utils/storage/utils";

const HIDDEN_STORAGE_KEY = "TRACES_FILTER_SIDEBAR_HIDDEN";

export const useFiltersSidebarVisible = () => {
  const { isMobile } = useDeviceDetect();
  const storageValue = getFromStorage(HIDDEN_STORAGE_KEY) === "true";
  const [isHidden, setIsHidden] = useState(isMobile ? true : storageValue);

  const setVisible = (isVisible: boolean) => {
    setIsHidden(!isVisible);
    if (isVisible) removeFromStorage([HIDDEN_STORAGE_KEY]);
    else saveToStorage(HIDDEN_STORAGE_KEY, "true");
  };

  const updateState = (e?: StorageEvent) => {
    if (e && e.key !== toPrefixedKey(HIDDEN_STORAGE_KEY)) return;
    // eslint-disable-next-line @eslint-react/set-state-in-effect -- reconciles hidden state with localStorage, which can only be read imperatively; also invoked directly from the "storage" event listener
    setIsHidden(getFromStorage(HIDDEN_STORAGE_KEY) === "true");
  };

  useEventListener("storage", updateState);
  useEffect(() => {
    if (!isMobile) updateState();
    // eslint-disable-next-line @eslint-react/exhaustive-deps -- intentionally runs once on mount to reconcile persisted state with the mount-time isMobile value, not on every later isMobile change
  }, []);

  return {
    isVisible: !isHidden,
    setVisible,
  };
};
