# typed: strict
# frozen_string_literal: true

class DestroyPageService < ApplicationService
  #   sig { params(form: PageDestroyingForm).void }
  #   def initialize(form:)
  #     @form = form
  #   end
  #
  #   sig { returns(Result) }
  #   def call
  #     if form.invalid?
  #       return Result.new(errors: errors_from_form(form))
  #     end
  #
  #     T.must(form.page).destroy!
  #
  #     Result.new(errors: [])
  #   end
  #
  #   private
  #
  #   sig { returns(PageDestroyingForm) }
  #   attr_reader :form
  #
  #   sig { params(form: PageDestroyingForm).returns(T::Array[Error]) }
  #   def errors_from_form(form)
  #     form.errors.map { |error| Error.new(message: error.full_message) }
  #   end
  #
  #   class Error < T::Struct
  #     const :message, String
  #   end
  #
  #   class Result < T::Struct
  #     const :errors, T::Array[Error]
  #   end
end
