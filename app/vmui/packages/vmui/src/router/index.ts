const router = {
  home: "/",
  trace: "/trace",
  icons: "/icons",
};

export interface RouterOptionsHeader {
  tenant?: boolean,
  timeSelector?: boolean,
  executionControls?: boolean,
}

export interface RouterOptions {
  title?: string,
  header: RouterOptionsHeader
}

export const routerOptions: { [key: string]: RouterOptions } = {
  [router.home]: {
    title: "Trace Explorer",
    header: {
      tenant: true,
      timeSelector: true,
      executionControls: true,
    }
  },
  [router.trace]: {
    title: "Trace ID",
    header: {
      tenant: true,
    }
  },
  [router.icons]: {
    title: "Icons",
    header: {}
  },
};

export default router;
