type NonotoConfig = {
  nonotoUrl: string;
  i18n: {
    messages: {
    };
  };
};

export const nonotoConfig = (window as any).NonotoConfig as NonotoConfig;
