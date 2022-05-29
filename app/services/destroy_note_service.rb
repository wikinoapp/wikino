# typed: strict
# frozen_string_literal: true

class DestroyNoteService < ApplicationService
  sig { params(form: NoteDestroyingForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    if form.invalid?
      return Result.new(errors: form.errors.full_messages.map { |message| Error.new(message:) })
    end

    T.must(form.note).destroy!

    Result.new(errors: [])
  end

  private

  sig { returns(NoteDestroyingForm) }
  attr_reader :form

  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error]
  end
end
