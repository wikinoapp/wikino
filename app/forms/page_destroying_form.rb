# typed: strict
# frozen_string_literal: true

class PageDestroyingForm < ApplicationForm
  sig { returns(T.nilable(User)) }
  attr_accessor :user

  sig { returns(T.nilable(::Page)) }
  attr_accessor :page

  validates :user, presence: true
  validates :page, presence: true
  validate :only_own_page_could_be_destroyed

  # @overload
  sig { returns(T::Boolean) }
  def persisted?
    true
  end

  sig { void }
  private def only_own_page_could_be_destroyed
    if user && page && !Pundit.policy(user, page).destroy?
      errors.add(:page, :only_own_page_could_be_destroyed)
    end
  end
end
