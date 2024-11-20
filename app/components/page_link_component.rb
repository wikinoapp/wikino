# typed: strict
# frozen_string_literal: true

class PageLinkComponent < ApplicationComponent
  sig { params(current_space: Space, page_location: PageLocation).void }
  def initialize(current_space:, page_location:)
    @current_space = T.let(current_space, Space)
    @page_location = T.let(page_location, PageLocation)
  end

  sig { returns(Space) }
  attr_reader :current_space
  private :current_space

  sig { returns(PageLocation) }
  attr_reader :page_location
  private :page_location

  sig { returns(String) }
  private def page_path
    "/s/#{current_space.identifier}/pages/#{page_location.page.number}"
  end
end
