import { useQuery, useMutation, UseQueryOptions, UseMutationOptions } from '@tanstack/solid-query';
import { apiRequest } from './api.service';
import { apiContract } from './contract';
import { z } from 'zod';

type Contract = typeof apiContract;

type Endpoints = {
  [R in keyof Contract]: {
    [A in keyof Contract[R]]: Contract[R][A];
  };
};

type HasResponse = { response: z.ZodType<any, any, any>; };
type HasBody = { body: z.ZodType<any, any, any>; };
type HasPath = { path: (id: any) => string; };

type Client<E extends Endpoints> = {
  [R in keyof E]: {
    [A in keyof E[R]]: E[R][A] extends { method: 'POST' | 'PUT' | 'PATCH' | 'DELETE' } & HasResponse ? 
      E[R][A] extends HasBody ?
        () => (options?: UseMutationOptions<z.infer<E[R][A]['response']>, unknown, z.infer<E[R][A]['body']>>) => 
          ReturnType<typeof useMutation<z.infer<E[R][A]['response']>, unknown, z.infer<E[R][A]['body']>>> :
        () => (options?: UseMutationOptions<z.infer<E[R][A]['response']>, unknown, void>) => 
          ReturnType<typeof useMutation<z.infer<E[R][A]['response']>, unknown, void>> :
      E[R][A] extends { method: 'GET' } & HasResponse ?
        E[R][A] extends HasPath ?
          (params: Parameters<E[R][A]['path']>[0], options?: UseQueryOptions<z.infer<E[R][A]['response']>>) => 
            ReturnType<typeof useQuery<z.infer<E[R][A]['response']>>> :
          (options?: UseQueryOptions<z.infer<E[R][A]['response']>>) => 
            ReturnType<typeof useQuery<z.infer<E[R][A]['response']>>> :
      never;
  };
};

function createApiClient(contract: Contract): Client<typeof contract> {
  const client = {} as Client<typeof contract>;

  for (const resource in contract) {
    client[resource] = {};
    for (const action in contract[resource]) {
      const endpoint = contract[resource][action];

      if (endpoint.method === 'POST' || endpoint.method === 'PUT' || endpoint.method === 'PATCH' || endpoint.method === 'DELETE') {
        client[resource][action] = () => (options) => {
          return useMutation({
            mutationFn: (data: any = {}) => {
              const path = typeof endpoint.path === 'function' ? endpoint.path(data.id) : endpoint.path;
              return apiRequest(endpoint.method, path, data);
            },
            ...options,
          });
        };
      } else {
        client[resource][action] = (params, options) => {
          const path = typeof endpoint.path === 'function' ? endpoint.path(params) : endpoint.path;
          return useQuery(() => ({ queryKey: [resource, action, params], queryFn: () => apiRequest(endpoint.method, path), ...options }));
        };
      }
    }
  }

  return client;
}

export const apiClient = createApiClient(apiContract);