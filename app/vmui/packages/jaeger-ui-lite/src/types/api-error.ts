export type ApiError =
  | string
  | {
      message: string;
      httpStatus?: number;
      httpStatusText?: string;
      httpUrl?: string;
      httpQuery?: string;
      httpBody?: string;
    };
