# typed: strict
# frozen_string_literal: true

class UpdatePageService < ApplicationService
  #   include PageUpsertable
  #
  #   sig { params(form: PageUpdatingForm).void }
  #   def initialize(form:)
  #     @form = form
  #   end
  #
  #   sig { returns(Result) }
  #   def call
  #     if form.invalid?
  #       return Result.new(page: nil, errors: errors_from_form(form))
  #     end
  #
  #     page = T.must(form.page)
  #     page.title = form.title
  #     page.modified_at = page.updated_at = Time.current
  #
  #     page_content = T.must(page.content)
  #     page_content.body = form.body
  #     page_content.body_html = form.body_html
  #
  #     page.save!
  #     page_content.save!
  #     page.link!
  #
  #     Result.new(page:, errors: [])
  #   end
  #
  #   private
  #
  #   sig { returns(PageUpdatingForm) }
  #   attr_reader :form
end
