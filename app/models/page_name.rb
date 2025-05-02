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
    Profile = new
    Settings = new
    SettingsProfile = new
    SignIn = new
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
