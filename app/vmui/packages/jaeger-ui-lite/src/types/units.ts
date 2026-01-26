/**
 * A branded type representing time values in microseconds.
 * This provides type safety to ensure time values are not confused with other numeric values.
 */
export type Microseconds = number & { readonly __brand: unique symbol };
