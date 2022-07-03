# typed: strict
# frozen_string_literal: true

class NoteDestroyingForm < ApplicationForm
  extend T::Sig

  sig { returns(T.nilable(User)) }
  attr_accessor :user

  sig { returns(T.nilable(::Note)) }
  attr_accessor :note

  validates :user, presence: true
  validates :note, presence: true
  validate :only_own_note_could_be_destroyed

  # @overload
  sig { returns(T::Boolean) }
  def persisted?
    true
  end

  sig { void }
  private def only_own_note_could_be_destroyed
    if user && note && !Pundit.policy(user, note).destroy?
      errors.add(:note, :only_own_note_could_be_destroyed)
    end
  end
end
