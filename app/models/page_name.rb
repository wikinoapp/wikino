# typed: strict
# frozen_string_literal: true

class PageName < T::Enum
  enums do
    Home = new
    Inbox = new
    PageDetail = new
    SignIn = new
    SpaceDetail = new
    TopicDetail = new
    Welcome = new
  end
end
