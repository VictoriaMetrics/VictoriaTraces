import { ComponentProps, FC, ReactNode } from "preact/compat";

type Props = { children: ReactNode };

export const combineComponents = (...components: FC<Props>[]): FC<Props> => {
  return components.reduce(
    (AccumulatedComponents, CurrentComponent) => {
      return ({ children }: ComponentProps<FC<Props>>): ReactNode => (
        <AccumulatedComponents>
          <CurrentComponent>{children}</CurrentComponent>
        </AccumulatedComponents>
      );
    },
    ({ children }) => <>{children}</>,
  );
};
