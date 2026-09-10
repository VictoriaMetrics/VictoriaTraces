import { isMobileAgent } from "../utils/detect-device";
import useWindowSize from "./useWindowSize";

export const getIsMobile = () => {
  const mobileAgent = isMobileAgent();
  const smallWidth = window.innerWidth < 500;
  return mobileAgent || smallWidth;
};

export default function useDeviceDetect() {
  useWindowSize();
  const isMobile = getIsMobile();

  return { isMobile };
}
