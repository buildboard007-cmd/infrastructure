import { StageEnvironment } from "../types/stage-environment";

export const GetAccountId = (e: StageEnvironment): string => {
  switch (e) {
    case StageEnvironment.DEV:
      return "521805123898";
    case StageEnvironment.PROD:
      return "186375394147";
  }

  return "000000000000";
};
