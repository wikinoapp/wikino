# typed: strict
# frozen_string_literal: true

module Forms
  class UpdateNoteForm < Forms::ApplicationForm
    include Forms::NoteUpsertable

    sig { returns(T.nilable(::Note)) }
    attr_reader :note

    validates :note, presence: true
    validate :title_should_be_unique

    sig { params(value: T.nilable(::Note)).void }
    def note=(value)
      @note = T.let(value, T.nilable(::Note))
    end

    # @overload
    sig { returns(T::Boolean) }
    def persisted?
      true
    end

    sig { void }
    private def title_should_be_unique
      return unless user
      return unless note

      notes = T.must(user).notes.where(title:)
      notes = notes.where.not(id: T.must(note).id)
      @original_note = notes.first

      if @original_note
        errors.add(:title, :title_should_be_unique)
      end
    end
  end
end
