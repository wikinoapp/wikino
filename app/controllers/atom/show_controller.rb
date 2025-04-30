# typed: true
# frozen_string_literal: true

module Atom
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      page_records = space_record.page_records.active.topics_visibility_public
        .order(published_at: :desc, id: :desc)
        .limit(15)

      space = SpaceRepository.new.to_model(space_record:)
      pages = page_records.map do |page_record|
        PageRepository.new.to_model(page_record:)
      end

      render(
        Atom::ShowView.new(space:, pages:),
        formats: :atom,
        content_type: Mime::Type.lookup("application/atom+xml")
      )
    end
  end
end
