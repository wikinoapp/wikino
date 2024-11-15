# typed: true
# frozen_string_literal: true

class ApplicationController < ActionController::Base
  extend T::Sig

  private def render_404
    render(
      file: Rails.public_path.join("404.html"),
      status: :not_found,
      layout: false,
      content_type: "text/html"
    )
  end
end
