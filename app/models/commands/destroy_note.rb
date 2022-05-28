# typed: strict
# frozen_string_literal: true

module Commands
  class DestroyNote
    extend T::Sig

    sig { params(form: Forms::NoteDestruction).void }
    def initialize(form:)
      @form = form
    end

    sig { returns(Result) }
    def run
      if form.invalid?
        return Result.new(errors: form.errors.full_messages.map { |message| Error.new(message:) })
      end

      T.must(form.note).destroy!

      Result.new(errors: [])
    end

    sig { returns(Forms::NoteDestruction) }
    private def form
      @form
    end

    class Error < T::Struct
      const :message, String
    end

    class Result < T::Struct
      const :errors, T::Array[Error]
    end
  end
end
