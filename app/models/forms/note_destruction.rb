# typed: strict
# frozen_string_literal: true

module Forms
  class NoteDestruction < Forms::Base
    extend T::Sig

    sig { returns(T.nilable(User)) }
    attr_reader :user

    sig { returns(T.nilable(::Note)) }
    attr_reader :note

    validates :user, presence: true
    validates :note, presence: true
    validate :only_own_note_could_be_destroyed

    sig { params(value: T.nilable(User)).void }
    def user=(value)
      @user = T.let(value, T.nilable(User))
    end

    sig { params(value: T.nilable(::Note)).void }
    def note=(value)
      @note = T.let(value, T.nilable(::Note))
    end

    # @overload
    sig { returns(T::Boolean) }
    def persisted?
      note&.persisted? == true
    end

    sig { void }
    private def only_own_note_could_be_destroyed
      if user && note && !Pundit.policy(user, note).destroy?
        errors.add(:note, :only_own_note_could_be_destroyed)
      end
    end
  end
end
