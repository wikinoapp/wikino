# typed: strict
# frozen_string_literal: true

class CreateInitialPageUseCase < ApplicationUseCase
  class Result < T::Struct
    const :page, Page
  end

  sig { params(topic: Topic).returns(Result) }
  def call(topic:)
    page = ActiveRecord::Base.transaction do
      new_page = Page.create_as_initial!(topic:)
      new_page.add_editor!(editor: Current.user!)
      new_page
    end

    Result.new(page:)
  end
end
