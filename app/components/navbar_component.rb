# typed: strict
# frozen_string_literal: true

class NavbarComponent < ApplicationComponent
  sig { params(signed_in: T::Boolean, current_page_name: PageName, class_name: String).void }
  def initialize(signed_in:, current_page_name:, class_name: "")
    @signed_in = signed_in
    @current_page_name = current_page_name
    @class_name = class_name
  end

  sig { returns(T::Boolean) }
  attr_reader :signed_in
  private :signed_in
  alias_method :signed_in?, :signed_in

  sig { returns(PageName) }
  attr_reader :current_page_name
  private :current_page_name

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

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
