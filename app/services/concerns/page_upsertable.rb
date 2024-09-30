# typed: strict
# frozen_string_literal: true

module PageUpsertable
  #   extend T::Sig
  #
  #   class Error < T::Struct
  #     const :message, String
  #   end
  #
  #   class DuplicatedPageError < T::Struct
  #     const :message, String
  #     const :original_page, T.nilable(Page)
  #   end
  #
  #   class Result < T::Struct
  #     const :page, T.nilable(Page)
  #     const :errors, T::Array[T.any(Error, DuplicatedPageError)]
  #   end
  #
  #   sig { params(form: T.any(PageCreatingForm, PageUpdatingForm)).returns(T::Array[T.any(Error, DuplicatedPageError)]) }
  #   def errors_from_form(form)
  #     form.errors.map do |error|
  #       if error.attribute == :title && error.type == :title_should_be_unique
  #         DuplicatedPageError.new(message: error.full_message, original_page: form.original_page)
  #       else
  #         Error.new(message: error.full_message)
  #       end
  #     end
  #   end
end
