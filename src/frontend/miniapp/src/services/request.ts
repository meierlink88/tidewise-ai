export interface RequestOptions<T> {
  mock: () => T | Promise<T>;
}

export async function request<T>(options: RequestOptions<T>): Promise<T> {
  return options.mock();
}
