# typed: strict
# frozen_string_literal: true

class SidebarComponent < ApplicationComponent
  sig do
    params(
      current_page_name: PageName,
      current_user: T.nilable(User),
      current_space: T.nilable(Space),
      variant: T.nilable(Symbol)
    ).void
  end
  def initialize(current_page_name:, current_user:, current_space:, variant: nil)
    @current_page_name = current_page_name
    @current_user = current_user
    @current_space = current_space
    @variant = T.let(variant || :fixed, Symbol)
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

  sig { returns(Symbol) }
  attr_reader :variant
  private :variant
end
