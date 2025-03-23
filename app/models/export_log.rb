# typed: strict
# frozen_string_literal: true

class ExportLog < ApplicationRecord
  belongs_to :space
  belongs_to :export
end
