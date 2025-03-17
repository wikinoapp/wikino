# typed: strict
# frozen_string_literal: true

class NavbarComponent < ApplicationComponent
  sig { params(current_page_name: PageName, current_user_entity: T.nilable(UserEntity), class_name: String).void }
  def initialize(current_page_name:, current_user_entity:, class_name: "")
    @current_page_name = current_page_name
    @current_user_entity = current_user_entity
    @class_name = class_name
  end

  sig { returns(PageName) }
  attr_reader :current_page_name
  private :current_page_name

  sig { returns(T.nilable(UserEntity)) }
  attr_reader :current_user_entity
  private :current_user_entity

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(T::Boolean) }
  private def signed_in?
    !current_user_entity.nil?
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
