import { PaginationResponseDto } from 'src/common/dto/pagination-response.dto';
import { TrackDto } from './track.dto';

export class SearchTrackResultDto extends PaginationResponseDto<TrackDto> {}
