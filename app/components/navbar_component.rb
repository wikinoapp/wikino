# typed: strict
# frozen_string_literal: true

class NavbarComponent < ApplicationComponent
  sig { params(current_page_name: PageName, current_user: T.nilable(User), current_space: T.nilable(Space), class_name: String).void }
  def initialize(current_page_name:, current_user:, current_space: nil, class_name: "")
    @current_page_name = current_page_name
    @current_user = current_user
    @current_space = current_space
    @class_name = class_name
  end

  sig { returns(PageName) }
  attr_reader :current_page_name
  private :current_page_name

  sig { returns(T.nilable(User)) }
  attr_reader :current_user
  private :current_user

  sig { returns(T.nilable(Space)) }
  attr_reader :current_space
  private :current_space

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
  private def search_icon_name
    (current_page_name == PageName::Search) ? "magnifying-glass-fill" : "magnifying-glass-regular"
  end

  sig { returns(String) }
  private def inbox_icon_name
    (current_page_name == PageName::Inbox) ? "tray-fill" : "tray-regular"
  end

  sig { returns(String) }
  private def profile_icon_name
    (current_page_name == PageName::Profile) ? "user-circle-fill" : "user-circle-regular"
  end

  sig { returns(String) }
  private def search_path_with_space_filter
    if current_space.present?
      search_path(q: "space:#{current_space.not_nil!.identifier}")
    else
      search_path
    end
  end
end
