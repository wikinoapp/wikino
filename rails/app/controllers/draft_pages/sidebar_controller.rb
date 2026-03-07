# typed: strict
# frozen_string_literal: true

module DraftPages
  class SidebarController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      result = DraftPageRepository.new.find_for_sidebar(
        user_record: current_user_record!,
        limit: 5
      )

      render_component(DraftPages::SidebarView.new(
        draft_pages: result[:draft_pages],
        has_more: result[:has_more]
      ))
    end
  end
end
