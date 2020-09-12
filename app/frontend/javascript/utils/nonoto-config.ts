type NonotoConfig = {
  nonotoUrl: string;
  i18n: {
    messages: {
      createNoteWithKeyword: string;
    };
  };
};

export const nonotoConfig = (window as any).NonotoConfig as NonotoConfig;
