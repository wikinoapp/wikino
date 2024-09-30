# typed: strict
# frozen_string_literal: true

class PageLink < ApplicationRecord
  belongs_to :space
  belongs_to :source_page, class_name: "Page"
  belongs_to :target_page, class_name: "Page"
end
