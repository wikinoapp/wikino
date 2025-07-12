# typed: strict
# frozen_string_literal: true

class PageName < T::Enum
  enums do
    AccountNew = new
    EmailConfirmationEdit = new
    Home = new
    Inbox = new
    PageDetail = new
    PageEdit = new
    PasswordEdit = new
    PasswordReset = new
    Profile = new
    Search = new
    Settings = new
    SettingsEmail = new
    SettingsProfile = new
    SettingsTwoFactorAuthDetail = new
    SettingsTwoFactorAuthNew = new
    SettingsTwoFactorAuthRecoveryCodes = new
    SignIn = new
    SignInTwoFactor = new
    SignInTwoFactorRecovery = new
    SignUp = new
    SpaceDetail = new
    SpaceNew = new
    SpaceSettings = new
    SpaceSettingsDeletion = new
    SpaceSettingsExportDetail = new
    SpaceSettingsExportsNew = new
    SpaceSettingsGeneral = new
    TopicDetail = new
    TopicEdit = new
    TopicNew = new
    TopicSettings = new
    TopicSettingsDeletion = new
    TopicSettingsGeneral = new
    Trash = new
    Welcome = new
  end
end
