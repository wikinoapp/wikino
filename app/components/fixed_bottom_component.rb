# typed: strict
# frozen_string_literal: true

class FixedBottomComponent < ApplicationComponent
  sig { params(current_page_name: PageName, current_user: T.nilable(User), current_space: T.nilable(Space), show_sidebar: T::Boolean).void }
  def initialize(current_page_name:, current_user:, current_space: nil, show_sidebar: true)
    @current_page_name = current_page_name
    @current_user = current_user
    @current_space = current_space
    @show_sidebar = show_sidebar
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

  sig { returns(T::Boolean) }
  attr_reader :show_sidebar
  private :show_sidebar
end
