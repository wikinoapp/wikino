# typed: strict
# frozen_string_literal: true

class NavbarComponent < ApplicationComponent
  sig { params(current_page_name: PageName, current_user: T.nilable(User), class_name: String).void }
  def initialize(current_page_name:, current_user:, class_name: "")
    @current_page_name = current_page_name
    @current_user = current_user
    @class_name = class_name
  end

  sig { returns(PageName) }
  attr_reader :current_page_name
  private :current_page_name

  sig { returns(T.nilable(User)) }
  attr_reader :current_user
  private :current_user

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(T::Boolean) }
  private def signed_in?
    !current_user.nil?
  end

  sig { returns(String) }
  private def home_icon_name
    (current_page_name == PageName::Home) ? "house-fill" : "house-regular"
  end

  sig { returns(String) }
  private def inbox_icon_name
    (current_page_name == PageName::Inbox) ? "tray-fill" : "tray-regular"
  end

  sig { returns(String) }
  private def profile_icon_name
    (current_page_name == PageName::Profile) ? "user-circle-fill" : "user-circle-regular"
  end
end
