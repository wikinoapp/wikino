# typed: strict
# frozen_string_literal: true

module Commands
  class CreateNote
    extend T::Sig

    sig { params(user: User, form: Forms::Note).void }
    def initialize(user:, form:)
      @user = user
      @form = form
    end

    sig { returns(Result) }
    def run
      if form.invalid?
        return Result.new(note: nil, errors: form.errors.full_messages.map { |message| Error.new(message:) })
      end

      modified_at = created_at = updated_at = Time.current
      note = user.notes.new(title: form.title, modified_at:, created_at:, updated_at:)
      note.build_content(user:, body: form.body)
      note.save!
      note.link!

      Result.new(note:, errors: [])
    end

    sig { returns(User) }
    private def user
      @user
    end

    sig { returns(Forms::Note) }
    private def form
      @form
    end

    class Error < T::Struct
      const :message, String
    end

    class Result < T::Struct
      const :note, T.nilable(Note)
      const :errors, T::Array[Error]
    end
  end
end
