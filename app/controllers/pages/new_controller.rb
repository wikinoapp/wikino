# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      authorize(Page.new, :new?)

      topic = Current.space!.topics.kept.find_by!(number: params[:topic_number])
      authorize(topic, :show?)

      result = CreateInitialPageUseCase.new.call(topic:)

      redirect_to edit_page_path(result.page.number)
    end
  end
end
