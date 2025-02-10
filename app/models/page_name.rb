# typed: strict
# frozen_string_literal: true

class PageName < T::Enum
  enums do
    Home = new
    Inbox = new
    PageDetail = new
    PageEdit = new
    SignIn = new
    SpaceDetail = new
    SpaceNew = new
    TopicDetail = new
    TopicEdit = new
    TopicNew = new
    Trash = new
    Welcome = new
  end
end
