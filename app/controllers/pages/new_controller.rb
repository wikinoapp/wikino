# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_topic

    sig { returns(T.untyped) }
    def call
      authorize(Page.new, :new?)

      result = CreateInitialPageUseCase.new.call(topic: @topic.not_nil!)

      redirect_to edit_page_path(Current.space!.identifier, result.page.number)
    end
  end
end
