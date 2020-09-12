# frozen_string_literal: true

class BacklinksComponent < ApplicationComponent
  def initialize(link_entities:)
    @link_entities = link_entities
  end

  private

  attr_reader :link_entities
end
