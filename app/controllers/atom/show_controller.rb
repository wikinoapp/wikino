# typed: true
# frozen_string_literal: true

module Atom
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      pages = space.pages.active
        .joins(:topic).merge(Topic.visibility_public)
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
