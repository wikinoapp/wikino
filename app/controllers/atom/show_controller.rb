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
      pages = space_record.page_records.active
        .joins(:topic_record).merge(TopicRecord.visibility_public)
        .order(published_at: :desc, id: :desc)
        .limit(15)

      render(
        Atom::ShowView.new(space:, pages:),
        formats: :atom,
        content_type: Mime::Type.lookup("application/atom+xml")
      )
    end
  end
end
