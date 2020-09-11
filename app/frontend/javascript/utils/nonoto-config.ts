type NonotoConfig = {
  nonotoUrl: string;
};

export const nonotoConfig = (window as any).NonotoConfig as NonotoConfig;
