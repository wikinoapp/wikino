# typed: strict
# frozen_string_literal: true

class DestroyNoteService < ApplicationService
  extend T::Sig

  sig { params(form: NoteDestroyingForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    if form.invalid?
      return Result.new(errors: errors_from_form(form))
    end

    T.must(form.note).destroy!

    Result.new(errors: [])
  end

  private

  sig { returns(NoteDestroyingForm) }
  attr_reader :form

  sig { params(form: NoteDestroyingForm).returns(T::Array[Error]) }
  def errors_from_form(form)
    form.errors.map { |error| Error.new(message: error.full_message) }
  end

  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error]
  end
end
