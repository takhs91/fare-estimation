package main

import (
    "encoding/csv"
    "fmt"
    "io"
    "os"
    "log"
    "time"
    "strconv"
    "github.com/umahmood/haversine"
)

// A Segment of two consecitive Records (points)
type Segment struct {
    U float64
    DeltaS float64
    DeltaT time.Duration
    T1 time.Time
    T2 time.Time
}

func (s Segment) String() string {
	return fmt.Sprintf("U: %v, DeltaS: %v, DeltaT: %v, T1: %v, T2: %v", s.U, s.DeltaS, s.DeltaT, s.T1, s.T2)
}

// A Record is a row of the csv and represents the position of the taxi at a given point
type Record struct {
    ID int
    Lat float64
    Lng float64
    Timestamp int64
}

func (r Record) String() string {
	return fmt.Sprintf("ID: %v, Lat: %v, Long: %v, Timestamp: %v", r.ID, r.Lat, r.Lng, r.Timestamp)
}

// StringArrayToRecord ransfroms an array of strings to a Record by pasing the strings accordingly
func StringArrayToRecord(array []string) (Record, error){
    var record Record
    var err error
    id, err := strconv.Atoi(array[0])
    if err != nil {
        return record, err
    }
    lat, err := strconv.ParseFloat(array[1], 64)
    if err != nil {
        return record, err
    }
    lng, err := strconv.ParseFloat(array[2], 64)
    if err != nil {
        return record, err
    }
    timestamp, err := strconv.ParseInt(array[3], 0, 64)
    if err != nil {
        return record, err
    }
    record = Record{id, lat, lng, timestamp}
    return record, err
}

// Flag is the the standard charge at the beggining of a ride
const Flag = 1.30
// MinimumFare is the minimum ride fare
const MinimumFare = 3.47
// IdleChargePerHour is the charge of staying Idle
const IdleChargePerHour = 11.90
// NormalChargePerKilometer is the charge during the day sift
const NormalChargePerKilometer = 0.74
// NightChargePerKilometer is the charge during the night sift
const NightChargePerKilometer = 1.30


// EstimateSegmentFare estimates the fare for one segment
func EstimateSegmentFare(segment *Segment) float64 {
    if segment.U <= 10.0 {
        return IdleChargePerHour * segment.DeltaT.Hours()
    }
    // thisDayStart := time.Date(segment.T1.Year(), segment.T1.Month(), segment.T1.Day(), 0, 0, 0, 0, time.Local)
    nextDayStart := time.Date(segment.T1.Year(), segment.T1.Month(), segment.T1.Day() + 1, 0, 0, 0, 0, time.Local)
    thisDayfiveMorning := time.Date(segment.T1.Year(), segment.T1.Month(), segment.T1.Day(), 5, 0, 0, 0, time.Local)
    switch {
        // Assume timestamps are SANE and t1 is no possible to be before
        // midnight and t2 after 5 o clock next day
        // If both measured times are in the day range
        case segment.T1.After(thisDayfiveMorning) && ( segment.T2.Before(nextDayStart) || segment.T2.Equal(nextDayStart) ):
            return NormalChargePerKilometer * segment.DeltaS
        // If both measured times are in the night range
        case ( segment.T1.Before(thisDayfiveMorning) || segment.T1.Equal(thisDayfiveMorning) ) && ( segment.T2.Before(thisDayfiveMorning) || segment.T2.Equal(thisDayfiveMorning) ):
            return NightChargePerKilometer * segment.DeltaS
        case segment.T1.After(thisDayfiveMorning) && segment.T2.After(nextDayStart) :
            dayRatio := nextDayStart.Sub(segment.T1).Hours() / segment.DeltaT.Hours()
            nightRatio := segment.T2.Sub(nextDayStart).Hours() / segment.DeltaT.Hours()
            return  NormalChargePerKilometer * segment.DeltaS * dayRatio +  NightChargePerKilometer * segment.DeltaS * nightRatio
        case (segment.T1.Before(thisDayfiveMorning) || segment.T1.Equal(thisDayfiveMorning)) && segment.T2.After(thisDayfiveMorning):
            nightRatio := thisDayfiveMorning.Sub(segment.T1).Hours() / segment.DeltaT.Hours()
            dayRatio := segment.T2.Sub(thisDayfiveMorning).Hours() / segment.DeltaT.Hours()
            return  NormalChargePerKilometer * segment.DeltaS * dayRatio +  NightChargePerKilometer * segment.DeltaS * nightRatio
    }
    return 0.0
}


// EstimateFare loops a ride's segments and calculate total estimated fare
func EstimateFare(segments *[]Segment) float64 {
    if segments == nil {
        return 0.0
    }
    var fareEstimate float64
    fareEstimate = Flag
    // fmt.Println(fareEstimate)
    for _, segment:= range(*segments) {
        fareEstimate += EstimateSegmentFare(&segment)
        // fmt.Println("For segment ", i, " Fair Estimate: ", fareEstimate)
    }
    if fareEstimate < MinimumFare {
        fareEstimate = MinimumFare
    }
    return fareEstimate
}


// FareEstimate holds ID and estimated fares tuples
type FareEstimate struct {
    ID int
    Fare float64
}

// FareEstimateToStringArray turns a FareEstimate ojects to slice of strings
func (f FareEstimate) FareEstimateToStringArray() []string {
    var array []string
    array = append(array, strconv.Itoa(f.ID))
    array = append(array, strconv.FormatFloat(f.Fare, 'f', 4, 64))
    return array
}

func main() {
    // Init Arguments
    resultsFileName := "estimated_fares.csv"
    argsWithoutProg := os.Args[1:]
    if len(argsWithoutProg) < 1 || len(argsWithoutProg) > 2{
        fmt.Println("Usage: ", os.Args[0], " record_file [output_file]")
        os.Exit(1)
    }
    if len(argsWithoutProg) == 2 {
        resultsFileName = argsWithoutProg[1]
    }
    recordsFileName :=  argsWithoutProg[0]

    // Open records file
    file, err := os.Open(recordsFileName)
    if err != nil {
        log.Fatal(err)
    }
    // automatically call Close() at the end of current method
    defer file.Close()

    // Initalize CSV reader
    reader := csv.NewReader(file)

    lineCount := 0

    // Initalize booking cariables
    var rideFareEstimates []FareEstimate
    var segments []Segment
    var previousRecord *Record
    for {
        record, err := reader.Read()
        // end-of-file is fitted into err
        if err == io.EOF {
            // Compute fare for the last ride
            fare := EstimateFare(&segments)
            rideFareEstimates = append(rideFareEstimates, FareEstimate{ID:previousRecord.ID, Fare: fare})
            segments = nil
            break
        } else if err != nil {
            log.Fatal(err)
        }

        var currentRecord Record
        currentRecord, err = StringArrayToRecord(record)
        if err != nil {
            log.Fatalln("Malformed Record:", err)
        }
        // fmt.Println("Previous Record: ", previousRecord)
        // fmt.Println("Curent Record: ", currentRecord)
        if previousRecord == nil {
            previousRecord = &currentRecord
            lineCount++
            continue
        }
        if previousRecord.ID == currentRecord.ID {
            coord1 := haversine.Coord{Lat: previousRecord.Lat, Lon: previousRecord.Lng}
            coord2 := haversine.Coord{Lat: currentRecord.Lat, Lon: currentRecord.Lng}
            _, deltaS := haversine.Distance(coord1, coord2)
            //  fmt.Println("Segment Kilometers:", deltaS)
            t1 := time.Unix(previousRecord.Timestamp,  0)
            // fmt.Println("Previous Record Time: ", t1)
            t2 := time.Unix(currentRecord.Timestamp, 0)
            // fmt.Println("Current Record Time: ", t2)
            deltaT := t2.Sub(t1)
            // fmt.Println("Delta: ", deltaT)
            u := deltaS / deltaT.Hours()
            // fmt.Println("Velocity: ", u)
            if u < 100.0 {
                segments = append(segments, Segment{U: u, DeltaS: deltaS, DeltaT: deltaT, T1:t1, T2:t2})
            } else {
                fmt.Println("Filtered line out ", lineCount, "with Record ", currentRecord, "found U ", u)
                lineCount++
                continue
            }
        } else {
            fare := EstimateFare(&segments)
            rideFareEstimates = append(rideFareEstimates, FareEstimate{ID:previousRecord.ID, Fare: fare})
            segments = nil
        }


        previousRecord = &currentRecord
        lineCount++
    }
    for _, v := range(rideFareEstimates) {
        fmt.Println("Ride with ID: ", v.ID, " Fare: ", v.Fare)
    }

    // Open the results file
	resultsFile, err := os.OpenFile(resultsFileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
        log.Fatal(err)
    }
    defer resultsFile.Close()

    writer := csv.NewWriter(resultsFile)
    for _, v := range(rideFareEstimates) {
        if err := writer.Write(v.FareEstimateToStringArray()); err != nil {
            log.Fatalln("error writing record to csv:", err)
		}
    }

	// Write any buffered data to the underlying writer.
	writer.Flush()

	if err := writer.Error(); err != nil {
        log.Fatal(err)
	}


}
